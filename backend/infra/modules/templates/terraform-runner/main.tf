# Terraform Runner Module - ECS Fargate for Terraform deployments

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.0"
    }
  }
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# Fetch available AZs if not provided
data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  # Use provided AZs or default to first 2 available in region
  availability_zones = coalesce(var.availability_zones, slice(data.aws_availability_zones.available.names, 0, 2))
}

