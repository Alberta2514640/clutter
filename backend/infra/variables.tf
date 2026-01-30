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
