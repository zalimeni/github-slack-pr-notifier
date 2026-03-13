package slack

import (
	"strconv"

	"github.com/zalimeni/github-slack-pr-notifier/internal/model"
)

type WorkflowPayload struct {
	EventType       string `json:"event_type"`
	EventLabel      string `json:"event_label"`
	Repo            string `json:"repo"`
	RepoURL         string `json:"repo_url"`
	PRNumber        string `json:"pr_number"`
	PRTitle         string `json:"pr_title"`
	PRURL           string `json:"pr_url"`
	Actor           string `json:"actor"`
	ActorURL        string `json:"actor_url"`
	ActionURL       string `json:"action_url"`
	Author          string `json:"author"`
	ReviewState     string `json:"review_state"`
	CommentExcerpt  string `json:"comment_excerpt"`
	FilePath        string `json:"file_path"`
	MentionDetected string `json:"mention_detected"`
	Reason          string `json:"reason"`
	UpdatedAt       string `json:"updated_at"`
	ThreadURL       string `json:"thread_url"`
	ThreadType      string `json:"thread_type"`
}

func NewPayload(notification model.Notification) WorkflowPayload {
	return WorkflowPayload{
		EventType:       notification.EventType,
		EventLabel:      notification.EventLabel,
		Repo:            notification.Repo,
		RepoURL:         notification.RepoURL,
		PRNumber:        strconv.Itoa(notification.PRNumber),
		PRTitle:         notification.PRTitle,
		PRURL:           notification.PRURL,
		Actor:           notification.Actor,
		ActorURL:        notification.ActorURL,
		ActionURL:       notification.ActionURL,
		Author:          notification.Author,
		ReviewState:     notification.ReviewState,
		CommentExcerpt:  notification.CommentExcerpt,
		FilePath:        notification.FilePath,
		MentionDetected: strconv.FormatBool(notification.MentionDetected),
		Reason:          notification.Reason,
		UpdatedAt:       notification.UpdatedAt,
		ThreadURL:       notification.ThreadURL,
		ThreadType:      notification.ThreadType,
	}
}
