terraform {
    required_providers {
        aws = {
            version = ">= 4.0.0"
            source  = "hashicorp/aws"
        }
        random = {
            source  = "hashicorp/random"
            version = "~> 3.0"
        }
    }
}

provider "aws" {
    region = var.aws_region
}