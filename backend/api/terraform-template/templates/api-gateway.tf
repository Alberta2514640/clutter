terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# API Gateway Module Template
# Variables: {api_name}, {jwt_authorizer_arn}, {aws_region}

# API Gateway REST API
resource "aws_api_gateway_rest_api" "api" {
  name        = var.api_name
  description = "API Gateway for ${var.api_name}"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}

# API Gateway JWT Authorizer
resource "aws_api_gateway_authorizer" "jwt-authorizer" {
  name                             = "jwt-authorizer"
  rest_api_id                      = aws_api_gateway_rest_api.api.id
  authorizer_uri                   = "arn:aws:apigateway:${var.aws_region}:lambda:path/2015-03-31/functions/${var.jwt_authorizer_arn}/invocations"
  authorizer_result_ttl_in_seconds = 300
  identity_source                  = "method.request.header.Authorization"
  type                             = "REQUEST"
}

# JWT Authorizer Lambda Permission
resource "aws_lambda_permission" "jwt_authorizer_permission" {
  statement_id  = "AllowExecutionFromAPIGatewayAuthorizer"
  action        = "lambda:InvokeFunction"
  function_name = var.jwt_authorizer_arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.api.execution_arn}/*/*"
}

# Request Validator
resource "aws_api_gateway_request_validator" "body-validator" {
  rest_api_id           = aws_api_gateway_rest_api.api.id
  name                  = "validate-body"
  validate_request_body = true
}
