output "rest_api_id" {
  description = "The ID of the Clutter API Gateway"
  value       = aws_api_gateway_rest_api.clutter-api.id
}
output "root_resource_id" {
  description = "The root resource ID of the Clutter API Gateway"
  value       = aws_api_gateway_rest_api.clutter-api.root_resource_id
}
output "execution_arn" {
  description = "The execution ARN of the Clutter API Gateway"
  value       = aws_api_gateway_rest_api.clutter-api.execution_arn
}
output "body_validator_id" {
  description = "The ID of the body validator rule"
  value       = aws_api_gateway_request_validator.body-validator.id
}

