package filter

import (
	"strings"

	"github.com/zalimeni/github-slack-pr-notifier/internal/config"
)

func AllowedRepo(cfg config.Config, repo string) bool {
	if len(cfg.RepoAllowlist) == 0 {
		return true
	}

	_, ok := cfg.RepoAllowlist[strings.ToLower(repo)]
	return ok
}

func Excerpt(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	const max = 220
	if len(body) <= max {
		return body
	}

	return body[:max-1] + "..."
}
