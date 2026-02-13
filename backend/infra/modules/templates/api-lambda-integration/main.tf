# Method definition
resource "aws_api_gateway_method" "method" {
  rest_api_id          = var.rest_api_id
  resource_id          = var.resource_id
  http_method          = var.http_method
  authorization        = var.jwt_authorizer_id != null ? "CUSTOM" : "NONE"
  authorizer_id        = var.jwt_authorizer_id
  request_validator_id = var.request_validator_id
  # if var.model_name is not null then request_models = var.model_name
  # otherwise request_models = empty map
  request_models = var.model_name != null ? {
    "application/json" = var.model_name
  } : {}
}
# Lambda Integration
resource "aws_api_gateway_integration" "lambda-integration" {
  rest_api_id             = var.rest_api_id
  resource_id             = var.resource_id
  http_method             = aws_api_gateway_method.method.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = var.invoke_arn
}
# Create a safe path part string for cases where path part from resource is using something like "{exampleID}"
locals {
  safe_path_part = replace(replace(var.path_part, "{", ""), "}", "")
}
# Permission for API Gateway to invoke the Lambda
resource "aws_lambda_permission" "lambda-permission" {
  statement_id  = "AllowExecutionFromAPIGateway-${local.safe_path_part}-${var.http_method}"
  action        = "lambda:InvokeFunction"
  function_name = var.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${var.execution_arn}/*/${var.http_method}${var.path}"
}
