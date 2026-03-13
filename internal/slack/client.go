package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/zalimeni/github-slack-pr-notifier/internal/model"
)

type Client struct {
	url        string
	httpClient *http.Client
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Send(ctx context.Context, notification model.Notification) error {
	payload, err := json.Marshal(NewPayload(notification))
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create slack request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("post to slack: %w", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts {
			if err := sleepWithContext(ctx, retryAfter); err != nil {
				return err
			}
			continue
		}

		return fmt.Errorf("slack returned %s: %s", resp.Status, string(body))
	}

	return fmt.Errorf("slack send exhausted retries")
}

func parseRetryAfter(raw string) time.Duration {
	if raw == "" {
		return time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return time.Second
	}
	return time.Duration(seconds) * time.Second
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
