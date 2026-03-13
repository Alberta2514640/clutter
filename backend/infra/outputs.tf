output "api_base_url" {
  description = "Base URL for the deployed API Gateway stage"
  value       = "https://${module.clutter-api-gateway.rest_api_id}.execute-api.${var.aws_region}.amazonaws.com/${var.stage_name}"
}

output "clutter_bucket_name" {
  description = "Name of the private Clutter bucket for Terraform and Ansible artifacts"
  value       = module.s3.clutter_bucket_name
}
