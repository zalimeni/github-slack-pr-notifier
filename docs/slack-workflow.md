# Slack workflow setup

Use `docs/workflow-example.json` as the best current exported example for this project.

Use `docs/assets/workflow-icon.png` as the workflow icon source image when you configure the workflow in Slack.

Important caveat:
- this still appears to be Slack's internal Workflow Builder serialization
- it is useful for adaptation and hand-editing against exported workflows
- it is still not a documented public import format
- if Slack import rejects it, use it as a reference and recreate the workflow in the UI

## Best workflow shape

This project now targets a 4-branch workflow shape reflected in `docs/workflow-example.json`:

- `review requested`
- `pull_request_review_comment`
- `issue_comment`
- `fallback`

The exported example preserves a real fallback branch from your Slack export, so this version is the best available reference.

## Variables expected by the notifier

Create a workflow with trigger `From a webhook` and make sure it accepts these text fields:

- `event_type`
- `event_label`
- `repo`
- `repo_url`
- `pr_number`
- `pr_title`
- `pr_url`
- `actor`
- `actor_url`
- `action_url`
- `author`
- `review_state`
- `comment_excerpt`
- `file_path`
- `mention_detected`
- `reason`
- `updated_at`
- `thread_url`
- `thread_type`

The exported Slack example only visibly uses some of these in message bodies, but the notifier payload includes all of them.

## Branch behavior

### Branch 1: review requested
Condition:
- `event_label == review requested`

Message shape:
```text
Review requested on repo PR #number: title
Requested by actor
Author: author
Open pull request
```

### Branch 2: inline comment
Condition:
- `event_type == pull_request_review_comment`

Message shape:
```text
Inline comment on repo PR #number: title
Actor: actor
File: file_path
Comment: comment_excerpt
Open comment
```

### Branch 3: PR comment or mention
Condition:
- `event_type == issue_comment`

Message shape:
```text
event_label on repo PR #number: title
Actor: actor
Comment: comment_excerpt
Open activity
```

### Branch 4: fallback
Condition:
- fallback case from the exported workflow structure

Message shape:
```text
event_label on repo PR #number: title
Author: author
Open activity
```

## Best reference file

- `docs/workflow-example.json`

## Regenerate the exported JSON

If you want to reuse the checked-in export with a different icon URL, title, description, or channel ID, run:

```bash
./scripts/generate-workflow-json.py \
  --icon-url https://raw.githubusercontent.com/zalimeni/github-slack-pr-notifier/main/docs/assets/workflow-icon.png \
  --channel-id C1234567890 \
  > docs/workflow-example.json
```

The script rewrites the exported workflow metadata in a deterministic way so you can keep a sanitized JSON example in git.

Use that file when comparing with a fresh export from Slack or when hand-editing an exported workflow.

## Testing

To test the live Slack workflow webhook directly:

```bash
SLACK_WORKFLOW_URL='https://hooks.slack.com/triggers/...' ./scripts/test-slack-workflow.sh
```

Or override values inline:

```bash
SLACK_WORKFLOW_URL='https://hooks.slack.com/triggers/...' \
EVENT_TYPE='pull_request_review_comment' \
EVENT_LABEL='inline comment' \
./scripts/test-slack-workflow.sh
```
