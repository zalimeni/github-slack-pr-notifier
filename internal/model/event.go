package model

import "time"

type Notification struct {
	EventType       string
	EventLabel      string
	Repo            string
	RepoURL         string
	PRNumber        int
	PRTitle         string
	PRURL           string
	Actor           string
	ActorURL        string
	ActionURL       string
	Author          string
	ReviewState     string
	CommentExcerpt  string
	FilePath        string
	MentionDetected bool
	Reason          string
	UpdatedAt       string
	UpdatedAtTime   time.Time
	ThreadURL       string
	ThreadType      string
	DedupKey        string
	DebounceKey     string
}
