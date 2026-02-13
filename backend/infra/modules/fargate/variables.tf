variable "s3_clutter" {
  description = "ARN of the S3 bucket for user Terraform/Ansible artifacts"
  type        = string
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
  default     = "latest"
}


