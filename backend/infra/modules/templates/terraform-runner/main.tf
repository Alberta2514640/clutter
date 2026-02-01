# =============================================================================
# Terraform Runner Module
# =============================================================================
# This module creates the infrastructure for running Terraform deployments
# on AWS Fargate. It includes:
# - VPC with public/private subnets and NAT Gateway
# - ECR repository for the Terraform runner Docker image
# - ECS cluster and task definition
# - IAM roles and policies
# - CloudWatch log group
# =============================================================================

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.0"
    }
  }
}

# Data source to get current AWS account ID
data "aws_caller_identity" "current" {}

# Data source to get current region
data "aws_region" "current" {}
