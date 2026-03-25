variable "s3_clutter_arn" {
  description = "ARN of the S3 bucket for user Terraform/Ansible artifacts"
  type        = string
}
variable "s3_clutter_name" {
  description = "Name of the S3 bucket for user Terraform/Ansible artifacts"
  type = string
}
variable "s3_templates_arn" {
  description = "ARN of the S3 bucket for common templates"
  type        = string
}
variable "s3_templates_name" {
  description = "Name of the S3 bucket for common templates"
  type = string
}

variable "s3_clutter_bucket_name" {
  description = "Name of the S3 Clutter bucket (for container environment variables)"
  type        = string
}

variable "aws_region" {
  description = "AWS region for container logging and environment configuration"
  type        = string
}

variable "ansible_runner_image_tag" {
  description = "Docker image tag for the Ansible runner container (avoid using 'latest' in production)"
  type        = string
}

variable "psql_connection_string" {
  description = "DEPRECATED: Secret is now managed outside Terraform. This variable is kept for backward compatibility but is no longer used."
  type        = string
  sensitive   = true
  default     = ""
}
