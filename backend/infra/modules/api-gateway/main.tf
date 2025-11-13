# API Gateway Resource
resource "aws_api_gateway_rest_api" "clutter-api" {

    name        = "clutter-api"
    description = <<EOT
    API Gateway for Clutter API endpoints.
    EOT

    endpoint_configuration {
        types   = ["REGIONAL"]
    }

}
# API Gateway request validators
# resource "aws_api_gateway_request_validator" "body_validator" {

#     rest_api_id           = aws_api_gateway_rest_api.clutter-api.id
#     name                  = "validate-body"
#     validate_request_body = true

# }
