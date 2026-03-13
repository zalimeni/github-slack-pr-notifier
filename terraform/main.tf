terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

locals {
  function_name       = var.project_name
  schedule_expression = var.poll_interval_minutes == 1 ? "rate(1 minute)" : "rate(${var.poll_interval_minutes} minutes)"
}
