output "model_name" {
  description = "The name of the model created"
  value       = aws_api_gateway_model.model.name
}

output "model_id" {
  description = "The id of the model created"
  value       = aws_api_gateway_model.model.id
}
