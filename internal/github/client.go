package github

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/zalimeni/github-slack-pr-notifier/internal/config"
	"github.com/zalimeni/github-slack-pr-notifier/internal/filter"
	"github.com/zalimeni/github-slack-pr-notifier/internal/model"
)

const baseURL = "https://api.github.com"

type Client struct {
	token      string
	httpClient *http.Client
}

type NotificationsResult struct {
	Threads      []Thread
	LastModified string
	PollInterval string
	NotModified  bool
}

type Thread struct {
	ID         string `json:"id"`
	Reason     string `json:"reason"`
	UpdatedAt  string `json:"updated_at"`
	LastReadAt string `json:"last_read_at"`
	Unread     bool   `json:"unread"`
	Subject    struct {
		Title            string `json:"title"`
		URL              string `json:"url"`
		LatestCommentURL string `json:"latest_comment_url"`
		Type             string `json:"type"`
	} `json:"subject"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
}

type PullRequest struct {
	Number             int    `json:"number"`
	Title              string `json:"title"`
	HTMLURL            string `json:"html_url"`
	ReviewCommentsURL  string `json:"review_comments_url"`
	CommentsURL        string `json:"comments_url"`
	User               User   `json:"user"`
	RequestedReviewers []User `json:"requested_reviewers"`
	RequestedTeams     []Team `json:"requested_teams"`
}

type Team struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	URL  string `json:"html_url"`
}

type ReviewComment struct {
	Path              string `json:"path"`
	Body              string `json:"body"`
	HTMLURL           string `json:"html_url"`
	User              User   `json:"user"`
	PullRequestReview struct {
		HTMLURL string `json:"html_url"`
	} `json:"pull_request_review"`
}

type IssueComment struct {
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	UpdatedAt string `json:"updated_at"`
	User      User   `json:"user"`
}

type User struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
}

func NewClient(token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{token: token, httpClient: httpClient}
}

func (c *Client) ListNotifications(ctx context.Context, cfg config.Config, lastModified string) (NotificationsResult, error) {
	values := url.Values{}
	values.Set("per_page", "50")
	values.Set("all", strconv.FormatBool(cfg.PollAll))
	values.Set("participating", strconv.FormatBool(cfg.PollParticipating))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/notifications?"+values.Encode(), nil)
	if err != nil {
		return NotificationsResult{}, fmt.Errorf("create notifications request: %w", err)
	}
	c.addHeaders(req)
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NotificationsResult{}, fmt.Errorf("list notifications: %w", err)
	}
	defer resp.Body.Close()

	result := NotificationsResult{
		LastModified: resp.Header.Get("Last-Modified"),
		PollInterval: resp.Header.Get("X-Poll-Interval"),
	}

	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return result, nil
	}
	if resp.StatusCode != http.StatusOK {
		return NotificationsResult{}, readError(resp, "list notifications")
	}

	if err := json.NewDecoder(resp.Body).Decode(&result.Threads); err != nil {
		return NotificationsResult{}, fmt.Errorf("decode notifications: %w", err)
	}

	return result, nil
}

func EnrichThread(ctx context.Context, client *Client, cfg config.Config, thread Thread) (model.Notification, bool, error) {
	if !filter.AllowedRepo(cfg, thread.Repository.FullName) {
		return model.Notification{}, false, nil
	}
	if thread.Subject.Type != "PullRequest" || thread.Subject.URL == "" {
		return model.Notification{}, false, nil
	}

	pr, err := client.GetPullRequest(ctx, thread.Subject.URL)
	if err != nil {
		return model.Notification{}, false, err
	}

	updatedAt := parseUpdatedAt(thread.UpdatedAt)
	notification := model.Notification{
		EventType:     mapReason(thread.Reason),
		EventLabel:    mapLabel(thread.Reason),
		Repo:          thread.Repository.FullName,
		RepoURL:       thread.Repository.HTMLURL,
		PRNumber:      pr.Number,
		PRTitle:       pr.Title,
		PRURL:         pr.HTMLURL,
		Author:        pr.User.Login,
		Reason:        thread.Reason,
		UpdatedAt:     thread.UpdatedAt,
		UpdatedAtTime: updatedAt,
		ThreadURL:     thread.Subject.URL,
		ThreadType:    thread.Subject.Type,
		ActionURL:     pr.HTMLURL,
	}

	switch thread.Reason {
	case "review_requested":
		directlyRequested, requestedTeam := reviewRequestTarget(cfg, pr)
		if !directlyRequested && requestedTeam == "" {
			log.Printf("notification action=skip_unallowed_review_request repo=%s pr=%d reason=%s requested_reviewers=%d requested_teams=%d", notification.Repo, notification.PRNumber, notification.Reason, len(pr.RequestedReviewers), len(pr.RequestedTeams))
			return model.Notification{}, false, nil
		}
		notification.RequestedTeam = requestedTeam
		if !finalizeNotification(&notification) {
			return model.Notification{}, false, nil
		}
		return notification, true, nil
	case "comment", "author", "subscribed", "manual", "mention", "team_mention", "state_change":
		latestURL := thread.Subject.LatestCommentURL
		if latestURL == "" {
			if !finalizeNotification(&notification) {
				return model.Notification{}, false, nil
			}
			return notification, true, nil
		}
		if strings.Contains(latestURL, "/pulls/comments/") {
			comment, err := client.GetReviewComment(ctx, latestURL)
			if err != nil {
				return model.Notification{}, false, err
			}
			notification.EventType = "pull_request_review_comment"
			notification.EventLabel = labelForCommentReason(thread.Reason, "inline comment")
			notification.Actor = comment.User.Login
			notification.ActorURL = comment.User.HTMLURL
			notification.ActionURL = comment.HTMLURL
			notification.CommentExcerpt = filter.Excerpt(comment.Body)
			notification.FilePath = comment.Path
			notification.MentionDetected = thread.Reason == "mention" || thread.Reason == "team_mention"
			if filter.IgnoreCommentActor(cfg, notification.Actor) {
				log.Printf("notification action=skip_ignored_actor repo=%s pr=%d event_type=%s actor=%q action_url=%q", notification.Repo, notification.PRNumber, notification.EventType, notification.Actor, notification.ActionURL)
				return model.Notification{}, false, nil
			}
			if !finalizeNotification(&notification) {
				return model.Notification{}, false, nil
			}
			return notification, true, nil
		}
		if strings.Contains(latestURL, "/issues/comments/") {
			comment, err := client.GetIssueComment(ctx, latestURL)
			if err != nil {
				return model.Notification{}, false, err
			}
			notification.EventType = "issue_comment"
			notification.EventLabel = labelForCommentReason(thread.Reason, "PR comment")
			notification.Actor = comment.User.Login
			notification.ActorURL = comment.User.HTMLURL
			notification.ActionURL = comment.HTMLURL
			notification.CommentExcerpt = filter.Excerpt(comment.Body)
			notification.MentionDetected = thread.Reason == "mention" || thread.Reason == "team_mention"
			if filter.IgnoreCommentActor(cfg, notification.Actor) {
				log.Printf("notification action=skip_ignored_actor repo=%s pr=%d event_type=%s actor=%q action_url=%q", notification.Repo, notification.PRNumber, notification.EventType, notification.Actor, notification.ActionURL)
				return model.Notification{}, false, nil
			}
			if !finalizeNotification(&notification) {
				return model.Notification{}, false, nil
			}
			return notification, true, nil
		}
		if !finalizeNotification(&notification) {
			return model.Notification{}, false, nil
		}
		return notification, true, nil
	default:
		return model.Notification{}, false, nil
	}
}

func (c *Client) GetPullRequest(ctx context.Context, apiURL string) (PullRequest, error) {
	var pr PullRequest
	if err := c.get(ctx, apiURL, &pr); err != nil {
		return PullRequest{}, err
	}
	return pr, nil
}

func (c *Client) GetIssueComment(ctx context.Context, apiURL string) (IssueComment, error) {
	var comment IssueComment
	if err := c.get(ctx, apiURL, &comment); err != nil {
		return IssueComment{}, err
	}
	return comment, nil
}

func (c *Client) GetReviewComment(ctx context.Context, apiURL string) (ReviewComment, error) {
	var comment ReviewComment
	if err := c.get(ctx, apiURL, &comment); err != nil {
		return ReviewComment{}, err
	}
	return comment, nil
}

func (c *Client) get(ctx context.Context, apiURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("create github request: %w", err)
	}
	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github get %s: %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readError(resp, "github get")
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}
	return nil
}

func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "github-slack-pr-notifier")
}

func readError(resp *http.Response, action string) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("%s failed with %s: %s", action, resp.Status, strings.TrimSpace(string(body)))
}

func mapReason(reason string) string {
	switch reason {
	case "review_requested":
		return "pull_request"
	case "comment", "author", "subscribed", "manual", "mention", "team_mention", "state_change":
		return "issue_comment"
	default:
		return reason
	}
}

func mapLabel(reason string) string {
	switch reason {
	case "review_requested":
		return "review requested"
	case "mention":
		return "mentioned on PR"
	case "team_mention":
		return "team mention on PR"
	case "state_change":
		return "PR state changed"
	case "author":
		return "activity on your PR"
	case "comment":
		return "PR comment"
	default:
		return strings.ReplaceAll(reason, "_", " ")
	}
}

func labelForCommentReason(reason, fallback string) string {
	switch reason {
	case "mention":
		return "mentioned on PR"
	case "team_mention":
		return "team mention on PR"
	case "state_change":
		return "PR state changed"
	case "author":
		return fallback
	case "comment":
		return fallback
	default:
		return fallback
	}
}

func parseRepoAndNumber(apiURL string) (string, int, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", 0, err
	}
	parts := strings.Split(strings.Trim(path.Clean(parsed.Path), "/"), "/")
	if len(parts) < 6 {
		return "", 0, fmt.Errorf("unexpected GitHub API URL: %s", apiURL)
	}
	number, err := strconv.Atoi(parts[5])
	if err != nil {
		return "", 0, fmt.Errorf("parse pull request number: %w", err)
	}
	return parts[1] + "/" + parts[2], number, nil
}

func finalizeNotification(notification *model.Notification) bool {
	if shouldSuppressNotification(*notification) {
		return false
	}
	notification.DedupKey = dedupKey(*notification)
	notification.DebounceKey = debounceKey(*notification)
	return true
}

func shouldSuppressNotification(notification model.Notification) bool {
	return notification.Reason == "author" &&
		notification.EventLabel == "activity on your PR" &&
		notification.Actor == "" &&
		notification.CommentExcerpt == "" &&
		notification.FilePath == ""
}

func dedupKey(notification model.Notification) string {
	switch notification.EventType {
	case "pull_request_review_comment", "issue_comment":
		return hashParts(
			notification.Repo,
			strconv.Itoa(notification.PRNumber),
			notification.EventType,
			notification.EventLabel,
			notification.Actor,
			notification.ActionURL,
		)
	case "pull_request":
		return hashParts(
			notification.Repo,
			strconv.Itoa(notification.PRNumber),
			notification.EventType,
			notification.EventLabel,
			notification.Reason,
		)
	default:
		return hashParts(
			notification.Repo,
			strconv.Itoa(notification.PRNumber),
			notification.EventType,
			notification.EventLabel,
			notification.Actor,
			notification.ActionURL,
			notification.ThreadURL,
			notification.Reason,
		)
	}
}

func debounceKey(notification model.Notification) string {
	switch notification.EventType {
	case "pull_request_review_comment", "issue_comment":
		return hashParts(
			notification.Repo,
			strconv.Itoa(notification.PRNumber),
			notification.EventType,
			notification.EventLabel,
			notification.ActionURL,
		)
	case "pull_request":
		return hashParts(
			notification.Repo,
			strconv.Itoa(notification.PRNumber),
			notification.EventType,
			notification.EventLabel,
			notification.Reason,
		)
	default:
		return hashParts(
			notification.Repo,
			strconv.Itoa(notification.PRNumber),
			notification.EventType,
			notification.EventLabel,
			notification.ThreadURL,
			notification.Reason,
		)
	}
}

func hashParts(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func parseUpdatedAt(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func reviewRequestTarget(cfg config.Config, pr PullRequest) (bool, string) {
	for _, reviewer := range pr.RequestedReviewers {
		if strings.EqualFold(strings.TrimSpace(reviewer.Login), cfg.GitHubUsername) {
			return true, ""
		}
	}

	for _, team := range pr.RequestedTeams {
		if filter.AllowTeamReviewRequest(cfg, team.Slug) {
			return false, team.Slug
		}
	}

	return false, ""
}
