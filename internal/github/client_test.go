package github

import (
	"testing"
	"time"

	"github.com/zalimeni/github-slack-pr-notifier/internal/config"
	"github.com/zalimeni/github-slack-pr-notifier/internal/filter"
	"github.com/zalimeni/github-slack-pr-notifier/internal/model"
)

func TestMapLabel(t *testing.T) {
	if got := mapLabel("review_requested"); got != "review requested" {
		t.Fatalf("unexpected label: %q", got)
	}
}

func TestDebounceKeyUsesActionURLForComments(t *testing.T) {
	base := model.Notification{
		EventType:      "issue_comment",
		EventLabel:     "PR comment",
		Repo:           "acme/repo",
		PRNumber:       42,
		Reason:         "comment",
		ActionURL:      "https://github.com/acme/repo/pull/42#issuecomment-1",
		CommentExcerpt: "first comment",
	}

	changedComment := base
	changedComment.CommentExcerpt = "different text"

	if got, want := debounceKey(base), debounceKey(changedComment); got != want {
		t.Fatalf("expected same debounce key for same comment URL, got %q and %q", got, want)
	}
}

func TestDebounceKeyDistinguishesDifferentCommentURLs(t *testing.T) {
	first := model.Notification{
		EventType:  "pull_request_review_comment",
		EventLabel: "inline comment",
		Repo:       "acme/repo",
		PRNumber:   42,
		Reason:     "comment",
		ActionURL:  "https://github.com/acme/repo/pull/42#discussion_r1",
	}

	second := first
	second.ActionURL = "https://github.com/acme/repo/pull/42#discussion_r2"

	if got, want := debounceKey(first), debounceKey(second); got == want {
		t.Fatalf("expected different debounce keys for different comment URLs, both were %q", got)
	}
}

func TestDedupKeyIgnoresUpdatedAtForReviewRequests(t *testing.T) {
	first := model.Notification{
		EventType:  "pull_request",
		EventLabel: "review requested",
		Repo:       "acme/repo",
		PRNumber:   42,
		Reason:     "review_requested",
		ActionURL:  "https://github.com/acme/repo/pull/42",
		UpdatedAt:  time.Unix(100, 0).UTC().Format(time.RFC3339),
	}

	second := first
	second.UpdatedAt = time.Unix(200, 0).UTC().Format(time.RFC3339)

	if got, want := dedupKey(first), dedupKey(second); got != want {
		t.Fatalf("expected same dedupe key for identical review request, got %q and %q", got, want)
	}
}

func TestDedupKeyIgnoresUpdatedAtForComments(t *testing.T) {
	first := model.Notification{
		EventType:  "issue_comment",
		EventLabel: "PR comment",
		Repo:       "acme/repo",
		PRNumber:   42,
		Actor:      "alice",
		ActionURL:  "https://github.com/acme/repo/pull/42#issuecomment-1",
		UpdatedAt:  time.Unix(100, 0).UTC().Format(time.RFC3339),
	}

	second := first
	second.UpdatedAt = time.Unix(200, 0).UTC().Format(time.RFC3339)

	if got, want := dedupKey(first), dedupKey(second); got != want {
		t.Fatalf("expected same dedupe key for identical comment, got %q and %q", got, want)
	}
}

func TestFinalizeNotificationSuppressesEmptyAuthorActivity(t *testing.T) {
	notification := model.Notification{
		EventType:  "issue_comment",
		EventLabel: "activity on your PR",
		Repo:       "acme/repo",
		PRNumber:   42,
		Reason:     "author",
		ActionURL:  "https://github.com/acme/repo/pull/42",
	}

	if finalizeNotification(&notification) {
		t.Fatal("expected empty author activity notification to be suppressed")
	}
}

func TestFinalizeNotificationKeepsEnrichedAuthorActivity(t *testing.T) {
	notification := model.Notification{
		EventType:      "issue_comment",
		EventLabel:     "PR comment",
		Repo:           "acme/repo",
		PRNumber:       42,
		Reason:         "author",
		ActionURL:      "https://github.com/acme/repo/pull/42#issuecomment-1",
		Actor:          "alice",
		CommentExcerpt: "looks good",
		UpdatedAt:      time.Unix(100, 0).UTC().Format(time.RFC3339),
		UpdatedAtTime:  time.Unix(100, 0).UTC(),
	}

	if !finalizeNotification(&notification) {
		t.Fatal("expected enriched author activity notification to be kept")
	}
	if notification.DedupKey == "" {
		t.Fatal("expected dedupe key to be populated")
	}
}

func TestShouldIgnoreCommentNotificationActor(t *testing.T) {
	if !filter.IgnoreCommentActor(config.Config{IgnoreGitHubActionsComments: true}, "github-actions[bot]") {
		t.Fatal("expected github-actions bot comments to be ignored")
	}
	if filter.IgnoreCommentActor(config.Config{IgnoreGitHubActionsComments: false}, "github-actions[bot]") {
		t.Fatal("expected github-actions bot comments to be allowed when disabled")
	}
}
