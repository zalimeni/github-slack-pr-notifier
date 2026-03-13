package filter

import (
	"testing"

	"github.com/zalimeni/github-slack-pr-notifier/internal/config"
)

func TestExcerpt(t *testing.T) {
	got := Excerpt("hello\n\nworld")
	if got != "hello world" {
		t.Fatalf("unexpected excerpt: %q", got)
	}
}

func TestIgnoreCommentActor(t *testing.T) {
	if !IgnoreCommentActor(config.Config{IgnoreGitHubActionsComments: true}, "github-actions[bot]") {
		t.Fatal("expected github-actions bot comments to be ignored by default")
	}
	if IgnoreCommentActor(config.Config{IgnoreGitHubActionsComments: false}, "github-actions[bot]") {
		t.Fatal("expected ignore flag to disable bot comment suppression")
	}
	if IgnoreCommentActor(config.Config{IgnoreGitHubActionsComments: true}, "alice") {
		t.Fatal("expected non-bot actor to be allowed")
	}
}

func TestAllowTeamReviewRequest(t *testing.T) {
	cfg := config.Config{TeamReviewRequestAllowlist: map[string]struct{}{"team-infragraph": {}}}

	if !AllowTeamReviewRequest(cfg, "team-infragraph") {
		t.Fatal("expected allowlisted team review request to be allowed")
	}
	if !AllowTeamReviewRequest(cfg, "TEAM-INFRAGRAPH") {
		t.Fatal("expected team slug matching to be case-insensitive")
	}
	if AllowTeamReviewRequest(cfg, "team-other") {
		t.Fatal("expected unlisted team review request to be blocked")
	}
	if AllowTeamReviewRequest(config.Config{}, "team-infragraph") {
		t.Fatal("expected team review requests to be blocked by default")
	}
}
