output "resource_id" {
  description = "The ID of the created API Gateway resource path"
  value       = aws_api_gateway_resource.path-resource.id
}
output "path_part" {
  description = "The path segment created for this API resource (e.g., log-in, diagram)"
  value       = aws_api_gateway_resource.path-resource.path_part
}
output "path" {
  description = "The full path of the created API resource (e.g., /diagram or /parent/child)"
  value       = aws_api_gateway_resource.path-resource.path
}
