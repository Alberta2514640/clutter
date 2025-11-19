variable "rest_api_id" {
  description = "Parent API Gateway ID"
  type        = string
}
variable "parent_id" {
  description = "The ID of the parent resouce that will contain this API endpoint (ex. /, /diagram, etc.)"
  type        = string
}
variable "path_part" {
  description = "API endpoint path name (ex. log-in)"
  type        = string
}
