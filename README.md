# github-slack-pr-notifier

Scheduled GitHub notifications poller that sends pull-request-related inbox activity into a Slack workflow webhook.

That is: A cheap replacement for GH PR notifications if your company won't enable enable the real thing. Only requires the ability to create a Slack channel workflow.

## What it does

- runs on an EventBridge schedule with a single Lambda function
- polls `GET /notifications` for the authenticated GitHub user with `If-Modified-Since`
- respects GitHub's notification-thread model instead of requiring repo or org webhooks
- filters to pull request threads only
- optionally filters to an allowlist of `org/repo`
- enriches the latest PR comment or inline review comment when GitHub provides a latest comment URL
- sends a flat JSON payload to a Slack workflow webhook
- loads the GitHub token and Slack workflow URL from AWS Secrets Manager at runtime
- stores the latest `Last-Modified` value plus dedupe/debounce fingerprints in DynamoDB

## GitHub auth requirements

Use a classic personal access token with:

- `notifications`
- `repo` for private repositories

GitHub documents the notifications API as poll-friendly and returns `Last-Modified`, `If-Modified-Since`, and `X-Poll-Interval` headers. This project polls every minute by default and persists the `Last-Modified` value between runs.

## Supported notification reasons

The poller currently emits Slack notifications for pull request threads with these GitHub reasons:

- `review_requested`
- `comment`
- `author`
- `mention`
- `team_mention`
- `subscribed`
- `manual`
- `state_change`

`review_requested` is sent only when the authenticated user is directly requested or a requested team slug is explicitly allowlisted. Reasons with a latest issue/review comment URL are enriched into either `PR comment` or `inline comment` messages.

## Required Lambda environment variables

- `SECRETS_MANAGER_ID`
- `GITHUB_USERNAME` (default Terraform value: `zalimeni`)
- `STATE_TABLE_NAME`

Optional:

- `REPO_ALLOWLIST` as comma-separated `org/repo`
- `TEAM_REVIEW_REQUEST_ALLOWLIST` as comma-separated GitHub team slugs, default empty
- `POLL_PARTICIPATING` default `true`
- `POLL_ALL` default `false`
- `IGNORE_GITHUB_ACTIONS_COMMENTS` default `true`

## Local build

```bash
go test ./...
go vet ./...
mkdir -p dist
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap ./cmd/poller-lambda
zip -j dist/function.zip bootstrap
```

## Terraform deploy

```bash
mkdir -p dist
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap ./cmd/poller-lambda
zip -j dist/function.zip bootstrap

cd terraform
terraform init
terraform apply \
  -var artifact_path=../dist/function.zip \
  -var github_token=ghp_xxx \
  -var slack_workflow_url=https://hooks.slack.com/triggers/... \
  -var repo_allowlist=org/repo-a,org/repo-b \
  -var team_review_request_allowlist=team-infragraph
```

Useful Terraform inputs:

- `aws_region` default `us-east-1`
- `github_username` default `zalimeni`
- `state_table_name` default `github-slack-pr-notifier-state`
- `team_review_request_allowlist` default empty
- `secrets_manager_id` default `github-slack-pr-notifier/runtime`
- `dedup_ttl` default `168h`
- `debounce_window` default `2m`
- `live_feed_window` default `10m`
- `poll_interval_minutes` default `1`
- `poll_participating` default `true`
- `poll_all` default `false`
- `ignore_github_actions_comments` default `true`

Outputs:

- `lambda_function_name`
- `schedule_expression`
- `state_table_name`

## Slack workflow setup

Create a workflow with:

- Trigger: `From a webhook`
- Slack workflow guidance and the best exported example: see `docs/slack-workflow.md` and `docs/workflow-example.json`
- Regenerate a customized exported workflow JSON with `./scripts/generate-workflow-json.py --icon-url https://raw.githubusercontent.com/zalimeni/github-slack-pr-notifier/main/docs/assets/workflow-icon.png --channel-id C1234567890 > docs/workflow-example.json`
- Use `docs/assets/workflow-icon.png` when setting the workflow icon in Slack
- Prefer the 4-branch workflow reflected there

## Deployment model

- EventBridge invokes Lambda every minute by default
- Lambda loads `Last-Modified` state and dedupe keys from DynamoDB
- Lambda calls GitHub notifications API with `If-Modified-Since`
- On `304 Not Modified`, Lambda exits
- On `200 OK`, Lambda enriches interesting PR threads and posts them to Slack
- Lambda stores the new `Last-Modified` value and sent-notification fingerprints in DynamoDB

## Current tradeoffs

- exact duplicates are suppressed with a fingerprint key in DynamoDB
- similar messages inside the debounce window are suppressed with a second rolling key
- only notifications updated within the live-feed window are eligible to send
- empty fallback `activity on your PR` notifications are suppressed unless GitHub gives enough comment context to enrich them
- `github-actions[bot]` PR and inline comment notifications are ignored by default
- team-originated review requests are ignored by default unless the team slug is allowlisted
- no mark-as-read behavior yet; this tool observes your inbox rather than mutating it
- no Slack threading because delivery uses a workflow webhook rather than a Slack app token
