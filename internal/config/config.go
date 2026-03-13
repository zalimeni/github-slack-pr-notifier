package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GitHubUsername    string
	StateTableName    string
	SecretsManagerID  string
	RepoAllowlist     map[string]struct{}
	PollParticipating bool
	PollAll           bool
	DedupTTL          time.Duration
	DebounceWindow    time.Duration
	LiveFeedWindow    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		GitHubUsername:    strings.ToLower(strings.TrimSpace(os.Getenv("GITHUB_USERNAME"))),
		StateTableName:    strings.TrimSpace(os.Getenv("STATE_TABLE_NAME")),
		SecretsManagerID:  strings.TrimSpace(os.Getenv("SECRETS_MANAGER_ID")),
		RepoAllowlist:     parseAllowlist(os.Getenv("REPO_ALLOWLIST")),
		PollParticipating: envBool("POLL_PARTICIPATING", true),
		PollAll:           envBool("POLL_ALL", false),
		DedupTTL:          envDuration("DEDUP_TTL", 7*24*time.Hour),
		DebounceWindow:    envDuration("DEBOUNCE_WINDOW", 2*time.Minute),
		LiveFeedWindow:    envDuration("LIVE_FEED_WINDOW", 10*time.Minute),
	}

	if cfg.GitHubUsername == "" {
		return Config{}, fmt.Errorf("GITHUB_USERNAME is required")
	}
	if cfg.StateTableName == "" {
		return Config{}, fmt.Errorf("STATE_TABLE_NAME is required")
	}
	if cfg.SecretsManagerID == "" {
		return Config{}, fmt.Errorf("SECRETS_MANAGER_ID is required")
	}
	if cfg.PollAll && cfg.PollParticipating {
		return Config{}, fmt.Errorf("POLL_ALL and POLL_PARTICIPATING cannot both be true")
	}

	return cfg, nil
}

func parseAllowlist(raw string) map[string]struct{} {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	allowed := make(map[string]struct{})
	for _, repo := range strings.Split(raw, ",") {
		repo = strings.ToLower(strings.TrimSpace(repo))
		if repo == "" {
			continue
		}
		allowed[repo] = struct{}{}
	}

	if len(allowed) == 0 {
		return nil
	}

	return allowed
}

func envBool(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}

	return value
}

func envDuration(key string, defaultValue time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return defaultValue
	}
	return value
}
