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
