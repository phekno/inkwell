output "domain" {
  value = "https://${var.domain}"
}

output "api_url" {
  value = "https://${var.domain}/api"
}

output "api_gateway_endpoint" {
  description = "Direct APIGW endpoint (no custom domain) — useful for debugging"
  value       = aws_apigatewayv2_api.http.api_endpoint
}

output "lambda_function_name" {
  value = aws_lambda_function.api.function_name
}

output "web_bucket" {
  value = aws_s3_bucket.web.bucket
}

output "web_cloudfront_domain" {
  value = aws_cloudfront_distribution.web.domain_name
}

output "web_cloudfront_id" {
  value = aws_cloudfront_distribution.web.id
}

output "cognito_user_pool_id" {
  value = aws_cognito_user_pool.this.id
}

output "cognito_web_client_id" {
  value = aws_cognito_user_pool_client.web.id
}

output "cognito_tui_client_id" {
  value = aws_cognito_user_pool_client.tui.id
}

