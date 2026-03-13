resource "aws_lambda_function" "this" {
  function_name = local.function_name
  role          = aws_iam_role.lambda.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["arm64"]
  timeout       = 60
  memory_size   = 256

  filename         = var.artifact_path
  source_code_hash = filebase64sha256(var.artifact_path)

  environment {
    variables = {
      GITHUB_USERNAME                = var.github_username
      REPO_ALLOWLIST                 = var.repo_allowlist
      TEAM_REVIEW_REQUEST_ALLOWLIST  = var.team_review_request_allowlist
      STATE_TABLE_NAME               = var.state_table_name
      SECRETS_MANAGER_ID             = aws_secretsmanager_secret.runtime.name
      POLL_PARTICIPATING             = tostring(var.poll_participating)
      POLL_ALL                       = tostring(var.poll_all)
      IGNORE_GITHUB_ACTIONS_COMMENTS = tostring(var.ignore_github_actions_comments)
      DEDUP_TTL                      = var.dedup_ttl
      DEBOUNCE_WINDOW                = var.debounce_window
      LIVE_FEED_WINDOW               = var.live_feed_window
    }
  }

  depends_on = [
    aws_iam_role_policy_attachment.lambda_basic,
    aws_iam_role_policy.lambda_runtime,
    aws_secretsmanager_secret.runtime,
  ]
}
