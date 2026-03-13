resource "aws_secretsmanager_secret" "runtime" {
  name = var.secrets_manager_id
}
