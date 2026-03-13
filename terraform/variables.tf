variable "aws_region" {
  description = "AWS region for Lambda deployment"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Lambda function name prefix"
  type        = string
  default     = "github-slack-pr-notifier"
}

variable "artifact_path" {
  description = "Path to the zipped Lambda artifact"
  type        = string
}

variable "secrets_manager_id" {
  description = "Secrets Manager secret name or ARN containing github_token and slack_workflow_url"
  type        = string
  default     = "github-slack-pr-notifier/runtime"
}

variable "github_username" {
  description = "GitHub username to notify for"
  type        = string
  default     = "zalimeni"
}

variable "repo_allowlist" {
  description = "Optional comma-separated allowlist of org/repo names"
  type        = string
  default     = ""
}

variable "team_review_request_allowlist" {
  description = "Optional comma-separated allowlist of team slugs whose review requests should notify"
  type        = string
  default     = ""
}

variable "state_table_name" {
  description = "DynamoDB table name used to persist Last-Modified state and dedupe records"
  type        = string
  default     = "github-slack-pr-notifier-state"
}

variable "poll_interval_minutes" {
  description = "EventBridge schedule interval in minutes"
  type        = number
  default     = 1
}

variable "poll_participating" {
  description = "Poll participating notifications"
  type        = bool
  default     = true
}

variable "poll_all" {
  description = "Poll all notifications instead of only participating"
  type        = bool
  default     = false
}

variable "ignore_github_actions_comments" {
  description = "Ignore comment notifications authored by github-actions[bot]"
  type        = bool
  default     = true
}

variable "dedup_ttl" {
  description = "How long processed notification fingerprints stay in storage"
  type        = string
  default     = "168h"
}

variable "debounce_window" {
  description = "Suppress similar notifications seen within this duration"
  type        = string
  default     = "2m"
}

variable "live_feed_window" {
  description = "Only notifications updated within this duration are eligible to send"
  type        = string
  default     = "10m"
}
