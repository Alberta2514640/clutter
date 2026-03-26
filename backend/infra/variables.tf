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
variable "terraform_output_bucket" {
  description = "S3 bucket name for Terraform engine output artifacts"
  type        = string
}
variable "terraform_template_bucket" {
  description = "S3 bucket name for Terraform engine templates"
  type        = string
}
variable "ansible_runner_image_tag" {
  description = "Docker image tag for the Ansible runner container (avoid using 'latest' in production)"
  type        = string
}
