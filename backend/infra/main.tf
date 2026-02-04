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
# ECS Fargate Task
# ================

module "fargate" {
  source = "./modules/fargate"

  s3_clutter = module.s3.clutter_bucket_arn
}

# ================
# Lambda Functions
# ================

# Log-in
module "log-in-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "log-in"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/log-in:*"
  ]
  zip_dir_slice = "log-in"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    JWT_SECRET             = var.jwt_secret
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

# Cloudformation URL Generator
module "cloudformation-stack-url-generator-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "cloudformation-stack-url-generator"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/cloudformation-stack-url-generator:*"
  ]
  zip_dir_slice = "cloudformation/stack-url-generator"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    CLOUDFORMATION_TEMPLATE_URL = var.cloudformation_template_url
    CLUTTER_ACCOUNT_ID = var.clutter_account_id
  }
}

# Organization
module "organization-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-create"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-create:*"
  ]
  zip_dir_slice = "organization/create"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "organization-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-get:*"
  ]
  zip_dir_slice = "organization/get"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "organization-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-update"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-update:*"
  ]
  zip_dir_slice = "organization/update"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "organization-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-delete"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-delete:*"
  ]
  zip_dir_slice = "organization/delete"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}

# Project
module "project-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-create"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/project-create:*"
  ]
  zip_dir_slice = "project/create"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "project-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/project-get:*"
  ]
  zip_dir_slice = "project/get"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "project-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-update"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/project-update:*"
  ]
  zip_dir_slice = "project/update"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "project-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "project-delete"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/project-delete:*"
  ]
  zip_dir_slice = "project/delete"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}

# Diagram
module "diagram-create-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-create"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/diagram-create:*"]
  zip_dir_slice = "diagram/create"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "diagram-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/diagram-get:*"]
  zip_dir_slice = "diagram/get"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "diagram-update-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-update"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/diagram-update:*"]
  zip_dir_slice = "diagram/update"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "diagram-delete-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "diagram-delete"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/diagram-delete:*"]
  zip_dir_slice = "diagram/delete"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}

# User Information
module "user-information-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "user-information-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/user-information-get:*"]
  zip_dir_slice = "user-information/get"
  environment_variables = {}

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

# Log-in
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

# CloudFormation Stack URL Generator
module "cloudformation-stack-url-generator-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "cloudformation-stack-url"
}

module "cloudformation-stack-url-generator-api-path-by-id" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.cloudformation-stack-url-generator-api-path.resource_id
  path_part   = "{organizationId}"
}

module "cloudformation-stack-url-generator-api-by-id-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.cloudformation-stack-url-generator-api-path-by-id.resource_id
  http_methods = ["GET"]
}

# Organization
module "organization-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "organization"
}
module "organization-api-path-by-id" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.organization-api-path.resource_id
  path_part   = "{organizationId}"
}
module "organization-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.organization-api-path.resource_id
  http_methods = ["POST", "GET"]
}
module "organization-api-by-id-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.organization-api-path-by-id.resource_id
  http_methods = ["PUT", "DELETE"]
}

# Project
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

# Diagram
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

# User Information
module "user-information-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "user-information"
}
module "user-information-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.user-information-api-path.resource_id
  http_methods = ["GET"]
}

# Validation Models

# Log-in
module "log-in-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "login"
  description     = "Model to validate log-in requests"
  schema_filename = "log-in.json"
}

# Organization
module "organization-create-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "organizationCreate"
  description     = "Model to validate organization creation requests"
  schema_filename = "organization-create.json"
}
# Organization
module "organization-update-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "organizationUpdate"
  description     = "Model to validate organization update requests"
  schema_filename = "organization-update.json"
}

# Project
module "project-create-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "projectCreate"
  description     = "Model to validate project creation requests"
  schema_filename = "project-create.json"
}
# Project
module "project-update-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "projectUpdate"
  description     = "Model to validate project update requests"
  schema_filename = "project-update.json"
}

# Diagram
module "diagram-create-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "diagramcreate"
  description     = "Model to validate diagram create requests"
  schema_filename = "diagram-create.json"
}
module "diagram-update-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "diagramupdate"
  description     = "Model to validate diagram update requests"
  schema_filename = "diagram-update.json"
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

# GET CloudFormation Stack URL (by organizationId)
module "cloudformation-stack-url-generator-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.cloudformation-stack-url-generator-api-path-by-id.resource_id
  http_method       = "GET"
  invoke_arn        = module.cloudformation-stack-url-generator-lambda.invoke_arn
  function_name     = module.cloudformation-stack-url-generator-lambda.function_name
  path_part         = module.cloudformation-stack-url-generator-api-path-by-id.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.cloudformation-stack-url-generator-api-path-by-id.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
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
# UPDATE organization
module "organization-update-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.organization-api-path-by-id.resource_id
  http_method          = "PUT"
  invoke_arn           = module.organization-update-lambda.invoke_arn
  function_name        = module.organization-update-lambda.function_name
  path_part            = module.organization-api-path-by-id.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.organization-api-path-by-id.path
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.organization-update-model.model_name
}
# DELETE organization
module "organization-delete-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-api-path-by-id.resource_id
  http_method       = "DELETE"
  invoke_arn        = module.organization-delete-lambda.invoke_arn
  function_name     = module.organization-delete-lambda.function_name
  path_part         = module.organization-api-path-by-id.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-api-path-by-id.path
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
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.project-create-model.model_name
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
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.project-update-model.model_name

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
  model_name           = module.diagram-create-model.model_name
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
}
# GET diagram
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
  model_name           = module.diagram-update-model.model_name
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

# User Information
# GET user-information
module "user-information-get-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.user-information-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.user-information-get-lambda.invoke_arn
  function_name     = module.user-information-get-lambda.function_name
  path_part         = module.user-information-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.user-information-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

// API Gateway Staging
resource "aws_api_gateway_deployment" "clutter" {
  rest_api_id = module.clutter-api-gateway.rest_api_id

  triggers = {
    redeploy = sha1(jsonencode([

      module.clutter-api-gateway.rest_api_id,
      module.clutter-api-gateway.jwt_authorizer_id,
      module.clutter-api-gateway.body_validator_id,

      module.log-in-model.model_id,
      module.organization-create-model.model_id,
      module.organization-update-model.model_id,
      module.project-create-model.model_id,
      module.project-update-model.model_id,
      module.diagram-create-model.model_id,
      module.diagram-update-model.model_id,

      module.log-in-api-integration.integration_id,

      module.cloudformation-stack-url-generator-api-integration.integration_id,

      module.organization-create-api-integration.integration_id,
      module.organization-get-api-integration.integration_id,
      module.organization-update-api-integration.integration_id,
      module.organization-delete-api-integration.integration_id,

      module.project-create-api-integration.integration_id,
      module.project-get-api-integration.integration_id,
      module.project-update-api-integration.integration_id,
      module.project-delete-api-integration.integration_id,

      module.diagram-create-api-integration.integration_id,
      module.diagram-get-api-integration.integration_id,
      module.diagram-update-api-integration.integration_id,
      module.diagram-delete-api-integration.integration_id,

      module.user-information-get-api-integration.integration_id
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

variable "stage_name" {
  type        = string
  description = "API Gateway stage name"
  default     = "prod"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.stage_name)
    error_message = "stage_name must be one of: dev, staging, prod."
  }
}

resource "aws_api_gateway_stage" "clutter" {
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  deployment_id = aws_api_gateway_deployment.clutter.id
  stage_name    = var.stage_name
}
