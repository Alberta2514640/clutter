variable "aws_region" {
  type        = string
  description = "The AWS region to deploy resources to"
}
variable "jwt_secret" {
  type        = string
  description = "JWT secret key"
}
variable "psql_connection_string" {
  type        = string
  description = "PostgreSQL connection string to enable querying"
}
variable "cloudformation_template_url" {
  description = "Full URL to the CloudFormation template stored in Clutter Templates S3"
  type        = string
}
variable "clutter_account_id" {
  description = "AWS Account ID for Clutter Fargate deployment role"
  type        = string
}
variable "frontend_url" {
  description = "Frontend URL for CORS allowed origin"
  type        = string
}
variable "ansible_runner_image_tag" {
  description = "Docker image tag for the Ansible runner container (avoid using 'latest' in production)"
  type        = string
}
variable "ansible_ssh_private_key" {
  description = "SSH private key PEM content for Ansible direct SSH transport"
  type        = string
  sensitive   = true
}
variable "ansible_ssh_known_hosts" {
  description = "known_hosts entries for strict SSH host verification"
  type        = string
  sensitive   = true
}
variable "ansible_target_ssh_cidrs" {
  description = "CIDR blocks that the Ansible runner may reach over SSH (port 22)"
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
