variable "aws_region" {
  type        = string
  description = "AWS region for deployment"
  default     = "us-east-1"
}

variable "ecs_cluster_name" {
  type        = string
  description = "Name of the ECS cluster"
  default     = "clutter-ecs-cluster"
}

variable "s3_bucket_name" {
  type        = string
  description = "S3 bucket name for terraform workspaces and logs"
}

variable "s3_bucket_arn" {
  type        = string
  description = "S3 bucket ARN for IAM policies"
}

variable "vpc_cidr" {
  type        = string
  description = "CIDR block for the VPC"
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  type        = list(string)
  description = "List of availability zones for subnets."
  default     = ["us-east-1a", "us-east-1b"]
}

variable "environment" {
  type        = string
  description = "Environment name (dev, staging, prod)"
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "environment must be one of: dev, staging, prod."
  }
}

variable "image_tag" {
  type        = string
  description = "Docker image tag for the terraform-runner container. Use explicit tags for reproducible deployments."
}

variable "cpu" {
  type        = number
  description = "Fargate task CPU units (256, 512, 1024, 2048, 4096)"
  default     = 512
}

variable "memory" {
  type        = number
  description = "Fargate task memory in MB"
  default     = 1024
}

variable "log_retention_days" {
  type        = number
  description = "CloudWatch log retention in days"
  default     = 30
}

variable "log_group_prefix" {
  type        = string
  description = "Prefix for CloudWatch log groups that the Terraform runner can manage"
  default     = "/aws/lambda/clutter-"
}
