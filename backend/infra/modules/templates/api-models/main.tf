# Model for validation
resource "aws_api_gateway_model" "model" {
  rest_api_id  = var.rest_api_id
  name         = var.model_name
  description  = var.description
  content_type = "application/json"
  schema       = file("${path.module}/schemas/${var.schema_filename}")
}
