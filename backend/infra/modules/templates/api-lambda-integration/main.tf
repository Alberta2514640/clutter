# Method definition
resource "aws_api_gateway_method" "method" {
  rest_api_id          = var.rest_api_id
  resource_id          = var.resource_id
  http_method          = var.http_method
  authorization        = "NONE"
  # if var.validator_id is not null then request_validator_id = var.request_validator_id
  # otherwise request_validator_id = null
  request_validator_id = var.request_validator_id != null ? var.request_validator_id : null
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
  integration_http_method = var.http_method
  type                    = "AWS_PROXY"
  uri                     = var.invoke_arn
}
# Permission for API Gateway to invoke the Lambda
resource "aws_lambda_permission" "lambda-permission" {
  statement_id  = "AllowExecutionFromAPIGateway-${var.path_part}"
  action        = "lambda:InvokeFunction"
  function_name = var.func_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${var.execution_arn}/*/${var.http_method}${var.path}"
}

# ==============================================================================
# CORS Compliance Below ========================================================
# ==============================================================================

# OPTIONS method for preflight request
resource "aws_api_gateway_method" "options-method" {
  rest_api_id   = var.rest_api_id
  resource_id   = var.resource_id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

# Mock integration to handle OPTIONS
resource "aws_api_gateway_integration" "options-integration" {
  rest_api_id = var.rest_api_id
  resource_id = var.resource_id
  http_method = aws_api_gateway_method.options_method.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

# OPTIONS method response
resource "aws_api_gateway_method_response" "options-response" {
  rest_api_id = var.rest_api_id
  resource_id = var.resource_id
  http_method = aws_api_gateway_method.options_method.http_method
  status_code = "200"

  response_models = {
    "application/json" = "Empty"
  }

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }
}

# Integration response with CORS headers
resource "aws_api_gateway_integration_response" "options-integration-response" {
  rest_api_id = var.rest_api_id
  resource_id = var.resource_id
  http_method = aws_api_gateway_method.options_method.http_method
  status_code = aws_api_gateway_method_response.options_response.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
    "method.response.header.Access-Control-Allow-Methods" = "'${var.http_method},OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}