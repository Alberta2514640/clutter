variable "s3_clutter_arn" {
  description = "ARN of the S3 bucket for user Terraform/Ansible artifacts"
  type        = string
}
variable "s3_clutter_name" {
  description = "Name of the S3 bucket for user Terraform/Ansible artifacts"
  type        = string
}
variable "s3_templates_arn" {
  description = "ARN of the S3 bucket for common templates"
  type        = string
}
variable "s3_templates_name" {
  description = "Name of the S3 bucket for common templates"
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
variable "ansible_ssh_private_key_secret_arn" {
  description = "Secrets Manager ARN containing the SSH private key PEM content for Ansible SSH transport"
  type        = string
}
variable "ansible_ssh_known_hosts_secret_arn" {
  description = "Secrets Manager ARN containing known_hosts entries for strict SSH host verification"
  type        = string
}
variable "ansible_target_ssh_cidrs" {
  description = "CIDR blocks that the Ansible runner is allowed to reach over SSH (port 22)"
  type        = list(string)
}
variable "ansible_remote_user" {
  description = "Default SSH username used by Ansible runner"
  type        = string
  default     = "ec2-user"
}
variable "ansible_ssh_host_address_source" {
  description = "Host address selection mode: public, private, public_or_private, private_or_public"
  type        = string
  default     = "private_or_public"

  validation {
    condition = contains(
      ["public", "private", "public_or_private", "private_or_public"],
      var.ansible_ssh_host_address_source
    )
    error_message = "ansible_ssh_host_address_source must be one of: public, private, public_or_private, private_or_public."
  }
}
variable "psql_connection_string" {
  description = "DEPRECATED: Secret is now managed outside Terraform. This variable is kept for backward compatibility but is no longer used."
  type        = string
  sensitive   = true
  default     = ""
}
