terraform {
  required_providers {
    aws = {
      version = ">= 4.0.0"
      source  = "hashicorp/aws"
    }
    random = {
      version = ">= 3.0.0"
      source  = "hashicorp/random"
    }
  }
}

provider "aws" {
  region = var.aws_region
}
