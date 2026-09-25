resource "aws_iam_role" "lambda" {
  name = "${local.name}-api"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = merge(local.app_tag, { Component = "api" })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "lambda_inline" {
  name = "${local.name}-api"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:Query",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
        ]
        Resource = aws_dynamodb_table.entries.arn
      },
      {
        Effect   = "Allow"
        Action   = ["kms:GenerateDataKey", "kms:Decrypt"]
        Resource = aws_kms_key.entries.arn
      },
    ]
  })
}

# Placeholder package — replaced by CI on every deploy.
data "archive_file" "placeholder" {
  type        = "zip"
  output_path = "${path.module}/.placeholder.zip"
  source {
    content  = "placeholder"
    filename = "bootstrap"
  }
}

resource "aws_lambda_function" "api" {
  function_name = "${local.name}-api"
  role          = aws_iam_role.lambda.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["arm64"]

  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256

  environment {
    variables = {
      ENTRIES_TABLE = aws_dynamodb_table.entries.name
      KMS_KEY_ID    = aws_kms_key.entries.key_id
    }
  }

  lifecycle {
    # CI updates the function code out-of-band; don't fight it from Terraform.
    ignore_changes = [filename, source_code_hash]
  }

  tags = merge(local.app_tag, { Component = "api" })
}

# Lambda auto-creates this group on first invoke — untagged and never
# expiring. Managing it here tags it for cost reports and caps retention.
import {
  to = aws_cloudwatch_log_group.api
  id = "/aws/lambda/${local.name}-api"
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/lambda/${local.name}-api"
  retention_in_days = 365

  tags = merge(local.app_tag, { Component = "api" })
}
