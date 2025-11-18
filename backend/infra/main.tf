# ================================
# Cognito User-Pools and Providers
# ================================
module "cognito" {
  source               = "./modules/templates/cognito"
  google_client_id     = var.google_client_id
  google_client_secret = var.google_client_secret
  frontend_url         = var.frontend_url
}
# ================
# Lambda Functions
# ================
module "diagram-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-create"
  actions       = ["dynamodb:PutItem"]
  resources     = ["*"]
  zip_dir_slice = "diagram/create"

}
module "diagram-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-get"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources     = ["*"]
  zip_dir_slice = "diagram/get"

}
module "diagram-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-update"
  actions = [
    "dynamodb:UpdateItem"
  ]
  resources     = ["*"]
  zip_dir_slice = "diagram/update"

}
module "diagram-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-delete"
  actions = [
    "dynamodb:DeleteItem"
  ]
  resources     = ["*"]
  zip_dir_slice = "diagram/delete"

}
# ===========
# API Gateway
# ===========
module "clutter-api-gateway" {
  source = "./modules/api-gateway"
}
# Paths
module "diagram-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "diagram"
}
module "diagram-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.diagram-api-path.resource_id
  http_methods = ["POST", "GET", "PUT", "DELETE"]
}
# Validation Models
module "test-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "test"
  description     = "Test model"
  schema_filename = "test.json"
}
# Integrations
# POST diagram
module "diagram-create-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.diagram-api-path.resource_id
  http_method          = "POST"
  invoke_arn           = module.diagram-create-lambda.invoke_arn
  function_name        = module.diagram-create-lambda.function_name
  path_part            = module.diagram-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.diagram-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.test-model.model_name
} # GET diagram
module "diagram-get-api-integration" {
  source        = "./modules/templates/api-lambda-integration"
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  resource_id   = module.diagram-api-path.resource_id
  http_method   = "GET"
  invoke_arn    = module.diagram-get-lambda.invoke_arn
  function_name = module.diagram-get-lambda.function_name
  path_part     = module.diagram-api-path.path_part
  execution_arn = module.clutter-api-gateway.execution_arn
  path          = module.diagram-api-path.path
}
# UPDATE diagram
module "diagram-update-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.diagram-api-path.resource_id
  http_method          = "PUT"
  invoke_arn           = module.diagram-update-lambda.invoke_arn
  function_name        = module.diagram-update-lambda.function_name
  path_part            = module.diagram-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.diagram-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.test-model.model_name
}
# DELETE diagram
module "diagram-delete-api-integration" {
  source        = "./modules/templates/api-lambda-integration"
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  resource_id   = module.diagram-api-path.resource_id
  http_method   = "DELETE"
  invoke_arn    = module.diagram-delete-lambda.invoke_arn
  function_name = module.diagram-delete-lambda.function_name
  path_part     = module.diagram-api-path.path_part
  execution_arn = module.clutter-api-gateway.execution_arn
  path          = module.diagram-api-path.path
}
