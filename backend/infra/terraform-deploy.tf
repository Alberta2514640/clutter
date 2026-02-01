# =============================================================================
# Terraform Deploy Infrastructure
# =============================================================================
# This file contains the infrastructure for running Terraform deployments
# on AWS Fargate, including:
# - ECS/Fargate infrastructure (cluster, task definition, ECR, VPC)
# - Lambda functions for deployment trigger, log writing, and log retrieval
# - API Gateway paths and integrations
# - S3 event notifications for log persistence
# =============================================================================

# ===================
# Terraform Runner Module
# ===================

module "terraform-runner" {
  source = "./modules/templates/terraform-runner"

  aws_region     = var.aws_region
  s3_bucket_name = module.s3.bucket_name
  s3_bucket_arn  = module.s3.bucket_arn
  environment    = var.stage_name
  image_tag      = var.terraform_runner_image_tag
}

# ===================
# Lambda Functions
# ===================

# Terraform Deploy Create Lambda
# Note: This Lambda uses a custom IAM role from the terraform-runner module
# because it needs ecs:RunTask and iam:PassRole permissions
resource "aws_lambda_function" "terraform-deploy-create" {
  function_name    = "terraform-deploy-create"
  role             = module.terraform-runner.deploy_trigger_lambda_role_arn
  handler          = "bootstrap"
  timeout          = 30
  filename         = "../api/terraform-deploy/create/deploy/bootstrap.zip"
  source_code_hash = fileexists("../api/terraform-deploy/create/deploy/bootstrap.zip") ? filebase64sha256("../api/terraform-deploy/create/deploy/bootstrap.zip") : null
  runtime          = "provided.al2"
  architectures    = ["arm64"]
  memory_size      = 128

  environment {
    variables = {
      PSQL_CONNECTION_STRING    = var.psql_connection_string
      ECS_CLUSTER_ARN           = module.terraform-runner.ecs_cluster_arn
      ECS_TASK_DEFINITION_ARN   = module.terraform-runner.task_definition_arn
      ECS_SUBNET_IDS            = join(",", module.terraform-runner.fargate_subnet_ids)
      ECS_SECURITY_GROUP_ID     = module.terraform-runner.security_group_id
      S3_BUCKET                 = module.s3.bucket_name
    }
  }
}

# Terraform Deploy Get Lambda (Log Retrieval)
module "terraform-deploy-get-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-deploy-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/terraform-deploy-get:*"]
  zip_dir_slice = "terraform-deploy/get"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

# Terraform Log Writer Lambda (S3 Triggered)
# Note: This Lambda uses a custom IAM role from the terraform-runner module
# because it needs s3:GetObject permissions on the logs prefix
resource "aws_lambda_function" "terraform-log-writer" {
  function_name    = "terraform-log-writer"
  role             = module.terraform-runner.log_writer_lambda_role_arn
  handler          = "bootstrap"
  timeout          = 60
  filename         = "../api/terraform-deploy/log-writer/deploy/bootstrap.zip"
  source_code_hash = fileexists("../api/terraform-deploy/log-writer/deploy/bootstrap.zip") ? filebase64sha256("../api/terraform-deploy/log-writer/deploy/bootstrap.zip") : null
  runtime          = "provided.al2"
  architectures    = ["arm64"]
  memory_size      = 256

  environment {
    variables = {
      PSQL_CONNECTION_STRING = var.psql_connection_string
    }
  }
}

# Lambda permission for S3 to invoke the log writer
resource "aws_lambda_permission" "s3_invoke_log_writer" {
  statement_id  = "AllowS3Invoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.terraform-log-writer.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = module.s3.bucket_arn
}

# S3 bucket notification for log writer
resource "aws_s3_bucket_notification" "log_writer_notification" {
  bucket = module.s3.bucket_id

  lambda_function {
    lambda_function_arn = aws_lambda_function.terraform-log-writer.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "logs/"
    filter_suffix       = ".log"
  }

  depends_on = [aws_lambda_permission.s3_invoke_log_writer]
}

# ===================
# API Gateway Paths
# ===================

# Deploy path (/deploy)
module "deploy-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "deploy"
}

module "deploy-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.deploy-api-path.resource_id
  http_methods = ["POST"]
}

# Deployment Log path (/deployment-log)
module "deployment-log-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "deployment-log"
}

# Deployment Log by ID path (/deployment-log/{runId})
module "deployment-log-by-id-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.deployment-log-api-path.resource_id
  path_part   = "{runId}"
}

module "deployment-log-by-id-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.deployment-log-by-id-api-path.resource_id
  http_methods = ["GET"]
}

# ===================
# API Gateway Validation Models
# ===================

module "deploy-create-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "deployCreate"
  description     = "Model to validate deployment trigger requests"
  schema_filename = "deploy-create.json"
}

# ===================
# API Gateway Integrations
# ===================

# POST /deploy
module "deploy-create-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.deploy-api-path.resource_id
  http_method          = "POST"
  invoke_arn           = aws_lambda_function.terraform-deploy-create.invoke_arn
  function_name        = aws_lambda_function.terraform-deploy-create.function_name
  path_part            = module.deploy-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.deploy-api-path.path
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.deploy-create-model.model_name
}

# Lambda permission for API Gateway to invoke terraform-deploy-create
resource "aws_lambda_permission" "apigw_invoke_deploy_create" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.terraform-deploy-create.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${module.clutter-api-gateway.execution_arn}/*/*"
}

# GET /deployment-log/{runId}
module "deployment-log-get-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.deployment-log-by-id-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.terraform-deploy-get-lambda.invoke_arn
  function_name     = module.terraform-deploy-get-lambda.function_name
  path_part         = module.deployment-log-by-id-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.deployment-log-by-id-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
