variable "domain" {
  description = "Custom domain for the web client and (path-based) API"
  type        = string
  default     = "journal.phekno.com"
}

variable "hosted_zone_name" {
  type    = string
  default = "phekno.com"
}

data "aws_route53_zone" "this" {
  name         = var.hosted_zone_name
  private_zone = false
}

# CloudFront requires us-east-1 for the cert; the stack is us-east-1 so the
# default provider works.
resource "aws_acm_certificate" "web" {
  domain_name       = var.domain
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.web.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  }

  zone_id         = data.aws_route53_zone.this.zone_id
  name            = each.value.name
  type            = each.value.type
  records         = [each.value.record]
  ttl             = 60
  allow_overwrite = true
}

resource "aws_acm_certificate_validation" "web" {
  certificate_arn         = aws_acm_certificate.web.arn
  validation_record_fqdns = [for r in aws_route53_record.cert_validation : r.fqdn]
}

# CloudFront Function: strip the /api prefix before sending the request to
# API Gateway so APIGW's routes ("GET /entries", not "GET /api/entries") match.
resource "aws_cloudfront_function" "strip_api_prefix" {
  name    = "${local.name}-strip-api-prefix"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<EOT
function handler(event) {
    var req = event.request;
    if (req.uri.startsWith('/api')) {
        req.uri = req.uri.slice(4);
        if (req.uri === '') { req.uri = '/'; }
    }
    return req;
}
EOT
}

# Rewrite extensionless paths (e.g. /entries) to /index.html so the SPA
# router can handle them. Scoped to the S3 default behavior so /api/*
# 404s pass through untouched.
resource "aws_cloudfront_function" "spa_fallback" {
  name    = "${local.name}-spa-fallback"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<EOT
function handler(event) {
    var req = event.request;
    var uri = req.uri;
    var lastSlash = uri.lastIndexOf('/');
    var lastSegment = uri.slice(lastSlash + 1);
    if (lastSegment.indexOf('.') === -1 && uri !== '/') {
        req.uri = '/index.html';
    }
    return req;
}
EOT
}

resource "aws_route53_record" "web_a" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = var.domain
  type    = "A"
  alias {
    name                   = aws_cloudfront_distribution.web.domain_name
    zone_id                = aws_cloudfront_distribution.web.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "web_aaaa" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = var.domain
  type    = "AAAA"
  alias {
    name                   = aws_cloudfront_distribution.web.domain_name
    zone_id                = aws_cloudfront_distribution.web.hosted_zone_id
    evaluate_target_health = false
  }
}
