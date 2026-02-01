# Core AWS settings
variable "aws_region" {
  type        = string
  description = "AWS region for deployment"
}

variable "availability_zones" {
  type        = list(string)
  description = "Availability zones for subnets (must match aws_region)"
  default     = null
}

# Networking
variable "vpc_cidr" {
  type        = string
  description = "CIDR block for the VPC"
  default     = "10.0.0.0/16"
}

# ECS configuration
variable "ecs_cluster_name" {
  type        = string
  description = "Name of the ECS cluster"
  default     = "clutter-ecs-cluster"
}

variable "environment" {
  type        = string
  description = "Environment name (dev, staging, prod)"
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Must be: dev, staging, or prod."
  }
}

variable "image_tag" {
  type        = string
  description = "Docker image tag for terraform-runner container"
}

variable "cpu" {
  type        = number
  description = "Fargate task CPU units"
  default     = 512

  validation {
    condition     = contains([256, 512, 1024, 2048, 4096], var.cpu)
    error_message = "Must be: 256, 512, 1024, 2048, or 4096."
  }
}

variable "memory" {
  type        = number
  description = "Fargate task memory in MB"
  default     = 1024

  validation {
    condition     = contains([512, 1024, 2048, 4096, 8192, 16384, 30720], var.memory)
    error_message = "Must be a valid Fargate memory value."
  }
}

# S3 configuration
variable "s3_bucket_name" {
  type        = string
  description = "S3 bucket for terraform workspaces and logs"
}

variable "s3_bucket_arn" {
  type        = string
  description = "S3 bucket ARN for IAM policies"
}

# Logging
variable "log_retention_days" {
  type        = number
  description = "CloudWatch log retention in days"
  default     = 30
}
