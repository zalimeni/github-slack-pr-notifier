#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_TOKEN:?GITHUB_TOKEN must be set}"
: "${SLACK_WORKFLOW_URL:?SLACK_WORKFLOW_URL must be set}"

SECRET_ID="${SECRET_ID:-github-slack-pr-notifier/runtime}"
AWS_REGION="${AWS_REGION:-us-east-1}"

TMP_JSON="$(mktemp)"
cleanup() {
  rm -f "$TMP_JSON"
}
trap cleanup EXIT

python3 - <<'PY' > "$TMP_JSON"
import json, os
print(json.dumps({
    'github_token': os.environ['GITHUB_TOKEN'],
    'slack_workflow_url': os.environ['SLACK_WORKFLOW_URL'],
}))
PY

aws secretsmanager put-secret-value \
  --region "$AWS_REGION" \
  --secret-id "$SECRET_ID" \
  --secret-string file://"$TMP_JSON" >/dev/null
