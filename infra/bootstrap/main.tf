# One-shot bootstrap: creates the S3 bucket and DynamoDB lock table that the
# root config in ../ uses as its remote backend.
#
# Apply this with LOCAL state, then move on:
#
#   cd infra/bootstrap
#   terraform init
#   terraform apply
#
# Then `cd ../ && terraform init` will pick up the backend below.

terraform {
  required_version = ">= 1.14.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.49"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "project" {
  type    = string
  default = "inkwell"
}

variable "github_repo" {
  description = "owner/repo for the GitHub OIDC trust"
  type        = string
  default     = "phekno/inkwell"
}

data "aws_caller_identity" "current" {}

locals {
  bucket_name = "${var.project}-tf-state-${data.aws_caller_identity.current.account_id}"
  table_name  = "${var.project}-tf-locks"
}

resource "aws_s3_bucket" "state" {
  bucket = local.bucket_name
}

resource "aws_s3_bucket_versioning" "state" {
  bucket = aws_s3_bucket.state.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "state" {
  bucket = aws_s3_bucket.state.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "state" {
  bucket                  = aws_s3_bucket.state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_dynamodb_table" "locks" {
  name         = local.table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }
}

output "bucket" {
  value = aws_s3_bucket.state.bucket
}

output "lock_table" {
  value = aws_dynamodb_table.locks.name
}

# ---------------------------------------------------------------------------
# GitHub Actions OIDC: provider + deploy role. Lives in the bootstrap so the
# main config in ../ can be applied by CI from the very first push.
# ---------------------------------------------------------------------------

# Account-scoped — reuse the existing GitHub OIDC provider rather than creating
# a duplicate (only one provider per URL is allowed per account).
data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

data "aws_iam_policy_document" "gh_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_repo}:ref:refs/heads/main",
        "repo:${var.github_repo}:pull_request",
      ]
    }
  }
}

resource "aws_iam_role" "gh_deploy" {
  name               = "${var.project}-gh-deploy"
  assume_role_policy = data.aws_iam_policy_document.gh_assume.json
}

# Broad managed policy is fine for a personal project; tighten later.
resource "aws_iam_role_policy_attachment" "gh_deploy_power" {
  role       = aws_iam_role.gh_deploy.name
  policy_arn = "arn:aws:iam::aws:policy/PowerUserAccess"
}

# IAM operations (creating/modifying roles for Lambda, etc.) need an extra grant.
resource "aws_iam_role_policy" "gh_deploy_iam" {
  name = "${var.project}-gh-deploy-iam"
  role = aws_iam_role.gh_deploy.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "iam:CreateRole", "iam:DeleteRole", "iam:GetRole", "iam:UpdateRole",
        "iam:PassRole", "iam:TagRole", "iam:UntagRole",
        "iam:AttachRolePolicy", "iam:DetachRolePolicy",
        "iam:PutRolePolicy", "iam:DeleteRolePolicy", "iam:GetRolePolicy",
        "iam:ListAttachedRolePolicies", "iam:ListRolePolicies",
      ]
      Resource = "*"
    }]
  })
}

output "gh_deploy_role_arn" {
  value = aws_iam_role.gh_deploy.arn
}
