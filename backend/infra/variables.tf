variable "aws_region" {
  type        = string
  description = "The AWS region to deploy resources to"
}
variable "google_client_id" {
  type        = string
  description = "Google OAuth client ID"
  sensitive   = true
}
variable "google_client_secret" {
  type        = string
  description = "Google OAuth client secret"
  sensitive   = true
}
variable "frontend_url" {
  type        = string
  description = "Base URL of the frontend application"
}

# ==================
# DynamoDB Variables
# ==================
variable "dynamodb_table_name" {
  type        = string
  description = "Name of the DynamoDB table"
  default     = "IaCDesigner"
}

variable "dynamodb_billing_mode" {
  type        = string
  description = "DynamoDB billing mode"
  default     = "PROVISIONED"
}

variable "dynamodb_enable_pitr" {
  type        = bool
  description = "Enable point-in-time recovery for DynamoDB"
  default     = true
}

variable "dynamodb_enable_streams" {
  type        = bool
  description = "Enable DynamoDB Streams"
  default     = false
}

# =============
# S3 Variables
# =============
variable "s3_enable_versioning" {
  type        = bool
  description = "Enable versioning for S3 bucket"
  default     = true
}

variable "s3_enable_cors" {
  type        = bool
  description = "Enable CORS for S3 bucket"
  default     = true
}

variable "s3_cors_origins" {
  type        = list(string)
  description = "Allowed origins for CORS"
  default     = ["*"]
}

# ===================
# General Variables
# ===================
variable "environment" {
  type        = string
  description = "Environment name (dev, staging, prod)"
  default     = "dev"
}

