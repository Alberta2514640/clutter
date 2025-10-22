# API Gateway Resource
resource "aws_api_gateway_rest_api" "clutter-api" {
    
    name        = "clutter-API"
    description = <<EOT
    API Gateway for Clutter endpoints.
    EOT

    endpoint_configuration {
        types   = ["REGIONAL"]
    }

}
# API Gateway endpoint paths
# ../diagram
resource "aws_api_gateway_resource" "diagram-path" {

    rest_api_id = aws_api_gateway_rest_api.clutter-api.id
    parent_id   = aws_api_gateway_rest_api.clutter-api.root_resource_id
    path_part   = "diagram"

}
