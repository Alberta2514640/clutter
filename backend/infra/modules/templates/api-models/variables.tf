variable "rest_api_id" {
  description = "Parent API Gateway ID"
  type        = string
}
variable "model_name" {
  description = "Name of the API Gateway model"
  type        = string
}
variable "description" {
  description = "Description of the model"
  type        = string
}
variable "schema_filename" {
  description = "JSON schema filename for validation"
  type        = string
}
