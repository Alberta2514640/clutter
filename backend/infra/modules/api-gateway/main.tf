# API Gateway Resource
resource "aws_api_gateway_rest_api" "clutter-api" {

  name        = "clutter-api"
  description = <<EOT
    API Gateway for Clutter API endpoints.
    EOT

  endpoint_configuration {
    types = ["REGIONAL"]
  }

}

# API Gateway JWT Authorizer
resource "aws_api_gateway_authorizer" "jwt-authorizer" {

  name                             = "jwt-authorizer"
  rest_api_id                      = aws_api_gateway_rest_api.clutter-api.id
  authorizer_uri                   = "arn:aws:apigateway:${var.aws_region}:lambda:path/2015-03-31/functions/${var.jwt_authorizer_arn}/invocations"
  authorizer_result_ttl_in_seconds = 0
  identity_source                  = "method.request.header.Authorization"
  type                             = "REQUEST"

}

# API Gateway JWT Authorizer invocation from API GW premission
resource "aws_lambda_permission" "jwt_authorizer_permission" {
  statement_id  = "AllowExecutionFromAPIGatewayAuthorizer"
  action        = "lambda:InvokeFunction"
  function_name = var.jwt_authorizer_arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.clutter-api.execution_arn}/*/*"
}


# Request validators
resource "aws_api_gateway_request_validator" "body-validator" {

  rest_api_id           = aws_api_gateway_rest_api.clutter-api.id
  name                  = "validate-body"
  validate_request_body = true

}
