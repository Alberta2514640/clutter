# ================
# Lambda Functions
# ================
module "diagram-create-lambda" {

  source = "./modules/templates/lambda"
  function_name = "diagram-create"
  actions = ["dynamodb:PutItem"]
  resources = ["*"]
  zip_dir_slice = "diagram/create"

}
module "diagram-get-lambda" {

  source = "./modules/templates/lambda"
  function_name = "diagram-get"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources = ["*"]
  zip_dir_slice = "diagram/get"

}
module "diagram-get-list-lambda" {

  source = "./modules/templates/lambda"
  function_name = "diagram-get-list"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources = ["*"]
  zip_dir_slice = "diagram/get-list"

}
module "diagram-update-lambda" {

  source = "./modules/templates/lambda"
  function_name = "diagram-update"
  actions = [
    "dynamodb:UpdateItem"
  ]
  resources = ["*"]
  zip_dir_slice = "diagram/update"

}
module "diagram-delete-lambda" {

  source = "./modules/templates/lambda"
  function_name = "diagram-delete"
  actions = [
    "dynamodb:DeleteItem"
  ]
  resources = ["*"]
  zip_dir_slice = "diagram/delete"

}
# ===========
# API Gateway
# ===========
module "clutter-api-gateway" {
    source = "./modules/api-gateway"
}
# Create API Gateway Validation Models
module "test-model" {
    source = "./modules/templates/api-models"
    rest_api_id = module.clutter-api-gateway.rest_api_id
    model_name = "test"
    description = "Test model"
    schema_filename = "test.json"
}
# Paths
module "diagram-create-api-path" {
  source = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id = module.clutter-api-gateway.root_resource_id
  path_part = "diagram"
}
# Integrations
module "diagram-reate-api-integration" {
  source = "./modules/templates/api-lambda-integration"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  resource_id = module.diagram-create-api-path.resource_id
  http_method = "POST"
  invoke_arn = module.diagram-create-lambda.invoke_arn
  function_name = module.diagram-create-lambda.function_name
  path_part = module.diagram-create-api-path.path_part
  execution_arn = module.clutter-api-gateway.execution_arn
  path = module.diagram-create-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name = module.test-model.model_name
}
