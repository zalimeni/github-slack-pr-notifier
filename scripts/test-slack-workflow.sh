#!/usr/bin/env bash
set -euo pipefail

: "${SLACK_WORKFLOW_URL:?SLACK_WORKFLOW_URL must be set}"

export EVENT_TYPE="${EVENT_TYPE:-pull_request_review_comment}"
export EVENT_LABEL="${EVENT_LABEL:-inline comment}"
export REPO="${REPO:-acme/service-api}"
export REPO_URL="${REPO_URL:-https://github.com/acme/service-api}"
export PR_NUMBER="${PR_NUMBER:-142}"
export PR_TITLE="${PR_TITLE:-Refactor auth middleware}"
export PR_URL="${PR_URL:-https://github.com/acme/service-api/pull/142}"
export ACTOR="${ACTOR:-alice}"
export ACTOR_URL="${ACTOR_URL:-https://github.com/alice}"
export ACTION_URL="${ACTION_URL:-https://github.com/acme/service-api/pull/142#discussion_r123456789}"
export AUTHOR="${AUTHOR:-zalimeni}"
export REVIEW_STATE="${REVIEW_STATE:-}"
export COMMENT_EXCERPT="${COMMENT_EXCERPT:-Can we avoid allocating here on every request?}"
export FILE_PATH="${FILE_PATH:-internal/auth/middleware.go}"
export MENTION_DETECTED="${MENTION_DETECTED:-false}"
export REASON="${REASON:-comment}"
export UPDATED_AT="${UPDATED_AT:-2026-03-12T20:10:12Z}"
export THREAD_URL="${THREAD_URL:-https://api.github.com/repos/acme/service-api/pulls/142}"
export THREAD_TYPE="${THREAD_TYPE:-PullRequest}"

python3 - <<'PY'
import json, os, urllib.request
url = os.environ['SLACK_WORKFLOW_URL']
payload = {
    'event_type': os.environ['EVENT_TYPE'],
    'event_label': os.environ['EVENT_LABEL'],
    'repo': os.environ['REPO'],
    'repo_url': os.environ['REPO_URL'],
    'pr_number': os.environ['PR_NUMBER'],
    'pr_title': os.environ['PR_TITLE'],
    'pr_url': os.environ['PR_URL'],
    'actor': os.environ['ACTOR'],
    'actor_url': os.environ['ACTOR_URL'],
    'action_url': os.environ['ACTION_URL'],
    'author': os.environ['AUTHOR'],
    'review_state': os.environ['REVIEW_STATE'],
    'comment_excerpt': os.environ['COMMENT_EXCERPT'],
    'file_path': os.environ['FILE_PATH'],
    'mention_detected': os.environ['MENTION_DETECTED'],
    'reason': os.environ['REASON'],
    'updated_at': os.environ['UPDATED_AT'],
    'thread_url': os.environ['THREAD_URL'],
    'thread_type': os.environ['THREAD_TYPE'],
}
req = urllib.request.Request(url, data=json.dumps(payload).encode('utf-8'), headers={'Content-Type': 'application/json'})
with urllib.request.urlopen(req, timeout=20) as resp:
    print(resp.status)
    print(resp.read().decode('utf-8', errors='replace'))
PY
