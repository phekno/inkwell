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

variable "web_domain" {
  description = "Optional custom domain for the web client (empty = use CloudFront default)"
  type        = string
  default     = ""
}
