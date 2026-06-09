terraform {
  required_version = ">= 1.14.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.49"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.6"
    }
  }

  # Populate `bucket` after running infra/bootstrap. The default form below
  # matches `inkwell-tf-state-<account_id>` in us-east-1.
  backend "s3" {
    key            = "inkwell/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "inkwell-tf-locks"
    encrypt        = true
    # bucket = "inkwell-tf-state-<account_id>"  # set via -backend-config in CI
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      Project   = "inkwell"
      ManagedBy = "terraform"
    }
  }
}
