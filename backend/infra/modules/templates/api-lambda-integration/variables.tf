# ================
# Method Variables
# ================
variable "rest_api_id" {
  description = "Parent API Gateway ID"
  type        = string
}
variable "resource_id" {
  description = "The ID of the path resource that will be the endpoint path itself (ex. /endpoint, /diagram/create)"
  type        = string
}
variable "http_method" {
  description = "HTTP method for the API Gateway method (e.g., GET, POST, PUT, DELETE, UPDATE)"
  type        = string
}
variable "request_validator_id" {
  description = "Validator ID for request body validation (optional)"
  type        = string
  default     = null
}
variable "model_name" {
  description = "Name of the API Gateway model (optional)"
  type        = string
  default     = null
}
variable "jwt_authorizer_id" {
  description = "The ID of the JWT authorizer"
  type        = string
  default     = null
}

# ============================
# Lambda Integration Variables
# ============================
variable "invoke_arn" {
  description = "Lambda invoke ARN"
  type        = string
}
variable "function_name" {
  description = "Lambda function name"
  type        = string
}
variable "path_part" {
  description = "API endpoint path name (ex. log-in)"
  type        = string
}
variable "execution_arn" {
  description = "Execution ARN of the REST API"
  type        = string
}
variable "path" {
  description = "The full path of the API resource (e.g., /diagram or /parent/child)"
  type        = string
}
