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
