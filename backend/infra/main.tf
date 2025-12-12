# ================================
# Cognito User-Pools and Providers
# ================================

module "cognito" {
  source               = "./modules/templates/cognito"
  google_client_id     = var.google_client_id
  google_client_secret = var.google_client_secret
  frontend_url         = var.frontend_url
}

# ========
# DynamoDB
# ========

module "dynamodb" {
  source = "./modules/dynamodb"
}

# ===
# S3
# ===

module "s3" {
  source = "./modules/s3"

  aws_region           = var.aws_region
  enable_versioning    = true
  enable_cors          = true
  cors_allowed_origins = ["*"]
}

# ================
# Lambda Functions
# ================

# Log-in
module "log-in-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "log-in"
  actions       = [
    "dynamodb:PutItem",
    "dynamodb:GetItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "log-in"
  environment_variables = {
    PSQL_CONNECTION_STRING  = var.psql_connection_string
    JWT_SECRET              = var.jwt_secret
  }

}

# Authorizer
module "authorizer-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "authorizer"
  actions       = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["*"]
  zip_dir_slice = "authorizer"
  environment_variables = {
    JWT_SECRET              = var.jwt_secret
  }

}

# Organization
module "organization-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-create"
  actions       = ["dynamodb:PutItem"]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "organization/create"
  environment_variables = {
    PSQL_CONNECTION_STRING  = var.psql_connection_string
  }

}
module "organization-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-get"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "organization/get"
  environment_variables = {
    PSQL_CONNECTION_STRING  = var.psql_connection_string
  }

}
module "organization-overview-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-overview"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "organization/overview"

}
module "organization-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-update"
  actions = [
    "dynamodb:UpdateItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "organization/update"

}
module "organization-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-delete"
  actions = [
    "dynamodb:DeleteItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "organization/delete"

}

# Project
module "project-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-create"
  actions       = ["dynamodb:PutItem"]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "project/create"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}
module "project-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-get"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "project/get"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}
module "project-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-update"
  actions = [
    "dynamodb:UpdateItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "project/update"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}
module "project-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-delete"
  actions = [
    "dynamodb:DeleteItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "project/delete"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}

# Diagram
module "diagram-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-create"
  actions       = ["dynamodb:PutItem"]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "diagram/create"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}
module "diagram-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-get"
  actions = [
    "dynamodb:GetItem",
    "dynamodb:Query",
    "dynamodb:Scan"
  ]
  resources     = [module.dynamodb.application_data_table_arn, module.dynamodb.supported_resurces_metadata_table_arn]
  zip_dir_slice = "diagram/get"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}
module "diagram-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-update"
  actions = [
    "dynamodb:UpdateItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "diagram/update"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}
module "diagram-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-delete"
  actions = [
    "dynamodb:DeleteItem"
  ]
  resources     = [module.dynamodb.application_data_table_arn]
  zip_dir_slice = "diagram/delete"
  environment_variables = {
    DDB_TABLE_NAME = var.ddb_application_table_name
  }

}

# ===========
# API Gateway
# ===========

module "clutter-api-gateway" {
  source = "./modules/api-gateway"
  aws_region = var.aws_region
  jwt_authorizer_arn = module.authorizer-lambda.arn
}

# Paths
module "log-in-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "log-in"
}
module "log-in-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.log-in-api-path.resource_id
  http_methods = ["POST"]
}
module "organization-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "organization"
}
module "organization-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.organization-api-path.resource_id
  http_methods = ["POST", "GET", "PUT", "DELETE"]
}
module "project-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "project"
}
module "project-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.project-api-path.resource_id
  http_methods = ["POST", "GET", "PUT", "DELETE"]
}
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
module "log-in-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "login"
  description     = "Model to validate log-in requests"
  schema_filename = "log-in.json"
}
module "organization-create-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "organizationCreate"
  description     = "Model to validate organization creation requests"
  schema_filename = "organization-create.json"
}

# Integrations
# POST Log-in
module "log-in-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.log-in-api-path.resource_id
  http_method          = "POST"
  invoke_arn           = module.log-in-lambda.invoke_arn
  function_name        = module.log-in-lambda.function_name
  path_part            = module.log-in-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.log-in-api-path.path

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.log-in-model.model_name
}

# Organization
# POST organization
module "organization-create-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.organization-api-path.resource_id
  http_method          = "POST"
  invoke_arn           = module.organization-create-lambda.invoke_arn
  function_name        = module.organization-create-lambda.function_name
  path_part            = module.organization-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.organization-api-path.path
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.organization-create-model.model_name
}
# GET organization
module "organization-get-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.organization-get-lambda.invoke_arn
  function_name     = module.organization-get-lambda.function_name
  path_part         = module.organization-api-path.path_part
  execution_arn      = module.clutter-api-gateway.execution_arn
  path              = module.organization-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# GET organization/overview
module "organization-overview-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.organization-api-path.resource_id
  path_part   = "overview"
}
module "organization-overview-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.organization-overview-api-path.resource_id
  http_methods = ["GET"]
}
module "organization-overview-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-overview-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.organization-overview-lambda.invoke_arn
  function_name     = module.organization-overview-lambda.function_name
  path_part         = module.organization-overview-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-overview-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# UPDATE organization
module "organization-update-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.organization-api-path.resource_id
  http_method          = "PUT"
  invoke_arn           = module.organization-update-lambda.invoke_arn
  function_name        = module.organization-update-lambda.function_name
  path_part            = module.organization-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.organization-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
}
# DELETE organization
module "organization-delete-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-api-path.resource_id
  http_method       = "DELETE"
  invoke_arn        = module.organization-delete-lambda.invoke_arn
  function_name     = module.organization-delete-lambda.function_name
  path_part         = module.organization-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

# Project
# POST project
module "project-create-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.project-api-path.resource_id
  http_method          = "POST"
  invoke_arn           = module.project-create-lambda.invoke_arn
  function_name        = module.project-create-lambda.function_name
  path_part            = module.project-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.project-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
}
# GET project
module "project-get-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.project-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.project-get-lambda.invoke_arn
  function_name     = module.project-get-lambda.function_name
  path_part         = module.project-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.project-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# UPDATE project
module "project-update-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.project-api-path.resource_id
  http_method          = "PUT"
  invoke_arn           = module.project-update-lambda.invoke_arn
  function_name        = module.project-update-lambda.function_name
  path_part            = module.project-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.project-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
}
# DELETE project
module "project-delete-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.project-api-path.resource_id
  http_method       = "DELETE"
  invoke_arn        = module.project-delete-lambda.invoke_arn
  function_name     = module.project-delete-lambda.function_name
  path_part         = module.project-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.project-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

# Diagram
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
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
} # GET diagram
module "diagram-get-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.diagram-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.diagram-get-lambda.invoke_arn
  function_name     = module.diagram-get-lambda.function_name
  path_part         = module.diagram-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.diagram-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
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
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
}
# DELETE diagram
module "diagram-delete-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.diagram-api-path.resource_id
  http_method       = "DELETE"
  invoke_arn        = module.diagram-delete-lambda.invoke_arn
  function_name     = module.diagram-delete-lambda.function_name
  path_part         = module.diagram-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.diagram-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
