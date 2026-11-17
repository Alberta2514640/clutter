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
variable "aws_region" {
  description = "AWS region for container logging and environment configuration"
  type        = string
}
variable "ansible_runner_image_tag" {
  description = "Version tag of the Ansible runner image (clutter-ansible-runner); never latest"
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$", var.ansible_runner_image_tag)) && var.ansible_runner_image_tag != "latest"
    error_message = "ansible_runner_image_tag must be a version tag such as v1, never latest."
  }
}
variable "terraform_deployer_image_tag" {
  description = "Version tag of the Terraform deployer image (clutter-terraform-deployer); never latest"
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$", var.terraform_deployer_image_tag)) && var.terraform_deployer_image_tag != "latest"
    error_message = "terraform_deployer_image_tag must be a version tag such as v1, never latest."
  }
}
