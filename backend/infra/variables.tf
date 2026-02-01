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
variable "ddb_application_table_name" {
    type        = string
    description = "DynamoDB application table name"
}
variable "jwt_secret" {
    type        = string
    description = "JWT secret key"
}
variable "psql_connection_string" {
    type        = string
    description = "PostgreSQL connection string to enable querying"
}

variable "terraform_runner_image_tag" {
    type        = string
    description = "Docker image tag for the terraform-runner container"
    default     = "latest"

    validation {
        condition     = can(regex("^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$", var.terraform_runner_image_tag))
        error_message = "Image tag must be 1-128 characters, start with alphanumeric, and contain only alphanumerics, dots, underscores, or hyphens."
    }
}
