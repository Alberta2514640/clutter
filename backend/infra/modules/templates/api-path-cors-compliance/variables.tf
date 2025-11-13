variable "rest_api_id" {
  description = "Parent API Gateway ID"
  type        = string
}
variable "resource_id" {
  description = "The ID of the path resource that will be the endpoint path itself (ex. /endpoint, /diagram/create)"
  type        = string
}
variable "http_methods" {
  description = "HTTP methods used for this particular API Gateway resource (e.g., [POST, PUT, DELETE])"
  type        = list(string)
}
