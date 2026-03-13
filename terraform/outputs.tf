output "lambda_function_name" {
  value = aws_lambda_function.this.function_name
}

output "schedule_expression" {
  value = aws_cloudwatch_event_rule.poller.schedule_expression
}

output "state_table_name" {
  value = aws_dynamodb_table.state.name
}

output "runtime_secret_name" {
  value = aws_secretsmanager_secret.runtime.name
}
