package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/zalimeni/github-slack-pr-notifier/internal/config"
	gh "github.com/zalimeni/github-slack-pr-notifier/internal/github"
	"github.com/zalimeni/github-slack-pr-notifier/internal/model"
	"github.com/zalimeni/github-slack-pr-notifier/internal/secrets"
	"github.com/zalimeni/github-slack-pr-notifier/internal/slack"
	"github.com/zalimeni/github-slack-pr-notifier/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	secretProvider, err := secrets.NewProvider(context.Background())
	if err != nil {
		log.Fatalf("init secrets provider: %v", err)
	}
	secretValues, err := secretProvider.GetSlackGitHubSecret(context.Background(), cfg.SecretsManagerID)
	if err != nil {
		log.Fatalf("load runtime secrets: %v", err)
	}

	httpClient := &http.Client{}
	githubClient := gh.NewClient(secretValues.GitHubToken, httpClient)
	slackClient := slack.NewClient(secretValues.SlackWorkflowURL)
	stateStore := store.NewDynamo(cfg.StateTableName)

	lambda.Start(func(ctx context.Context, _ events.CloudWatchEvent) (response, error) {
		return handle(ctx, cfg, githubClient, slackClient, stateStore)
	})
}

type response struct {
	Checked           int    `json:"checked"`
	Sent              int    `json:"sent"`
	SkippedDuplicate  int    `json:"skipped_duplicate"`
	SkippedDebounced  int    `json:"skipped_debounced"`
	SkippedStale      int    `json:"skipped_stale"`
	LastModified      string `json:"last_modified,omitempty"`
	PollInterval      string `json:"poll_interval,omitempty"`
	NotificationCount int    `json:"notification_count"`
}

func handle(ctx context.Context, cfg config.Config, githubClient *gh.Client, slackClient *slack.Client, stateStore *store.Dynamo) (response, error) {
	state, err := stateStore.LoadState(ctx)
	if err != nil {
		return response{}, err
	}

	result, err := githubClient.ListNotifications(ctx, cfg, state.LastModified)
	if err != nil {
		return response{}, err
	}

	if result.NotModified {
		return response{
			Checked:      1,
			LastModified: state.LastModified,
			PollInterval: result.PollInterval,
		}, nil
	}

	resp := response{
		Checked:           1,
		LastModified:      result.LastModified,
		PollInterval:      result.PollInterval,
		NotificationCount: len(result.Threads),
	}

	now := time.Now().UTC()
	for _, thread := range result.Threads {
		notification, ok, err := gh.EnrichThread(ctx, githubClient, cfg, thread)
		if err != nil {
			return response{}, err
		}
		if !ok {
			continue
		}
		if !isLive(notification, cfg, now) {
			resp.SkippedStale++
			continue
		}

		decision, err := shouldSend(ctx, cfg, stateStore, notification)
		if err != nil {
			return response{}, err
		}
		switch decision {
		case decisionDuplicate:
			resp.SkippedDuplicate++
			continue
		case decisionDebounced:
			resp.SkippedDebounced++
			continue
		}

		if err := slackClient.Send(ctx, notification); err != nil {
			return response{}, err
		}
		if err := recordSend(ctx, cfg, stateStore, notification); err != nil {
			return response{}, err
		}
		resp.Sent++
	}

	if result.LastModified == "" {
		result.LastModified = state.LastModified
	}
	if err := stateStore.SaveState(ctx, store.State{LastModified: result.LastModified}); err != nil {
		return response{}, err
	}
	resp.LastModified = result.LastModified
	return resp, nil
}

type sendDecision int

const (
	decisionSend sendDecision = iota
	decisionDuplicate
	decisionDebounced
)

func shouldSend(ctx context.Context, cfg config.Config, stateStore *store.Dynamo, notification model.Notification) (sendDecision, error) {
	seen, err := stateStore.Seen(ctx, notification.DedupKey)
	if err != nil {
		return decisionSend, err
	}
	if seen {
		return decisionDuplicate, nil
	}

	if notification.UpdatedAtTime.IsZero() {
		return decisionSend, nil
	}

	debounced, err := stateStore.SeenWithin(ctx, notification.DebounceKey, cfg.DebounceWindow, notification.UpdatedAtTime)
	if err != nil {
		return decisionSend, err
	}
	if debounced {
		return decisionDebounced, nil
	}

	return decisionSend, nil
}

func recordSend(ctx context.Context, cfg config.Config, stateStore *store.Dynamo, notification model.Notification) error {
	recordedAt := notification.UpdatedAtTime
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}
	if err := stateStore.Record(ctx, notification.DedupKey, recordedAt, cfg.DedupTTL); err != nil {
		return err
	}
	if err := stateStore.RecordWindow(ctx, notification.DebounceKey, recordedAt, cfg.DedupTTL); err != nil {
		return err
	}
	return nil
}

func (r response) String() string {
	payload, _ := json.Marshal(r)
	return string(payload)
}

func isLive(notification model.Notification, cfg config.Config, now time.Time) bool {
	if cfg.LiveFeedWindow <= 0 {
		return true
	}
	if notification.UpdatedAtTime.IsZero() {
		return true
	}
	return !notification.UpdatedAtTime.Before(now.Add(-cfg.LiveFeedWindow))
}
