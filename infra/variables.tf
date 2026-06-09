variable "region" {
  type    = string
  default = "us-east-1"
}

variable "project" {
  type    = string
  default = "inkwell"
}

variable "web_domain" {
  description = "Optional custom domain for the web client (empty = use CloudFront default)"
  type        = string
  default     = ""
}
