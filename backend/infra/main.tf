# ===
# S3
# ===

module "s3" {
  source = "./modules/s3"

  aws_region              = var.aws_region
  enable_versioning       = true
  enable_cors             = true
  cors_allowed_origins    = ["*"]
  ansible_runner_role_id  = module.fargate.ansible_task_role_unique_id
}

# ================
# ECS Fargate Task
# ================

module "fargate" {
  source = "./modules/fargate"

  s3_clutter                = module.s3.clutter_bucket_arn
  s3_clutter_bucket_name    = module.s3.clutter_bucket_name
  aws_region                = var.aws_region
  psql_connection_string    = var.psql_connection_string
  ansible_runner_image_tag  = var.ansible_runner_image_tag
}

# ========================
# Ansible Engine Resources
# ========================



# DynamoDB table for Terraform state locking
resource "aws_dynamodb_table" "terraform_state_lock" {
  name           = "terraform-state-lock"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Name        = "terraform-state-lock"
    Description = "Terraform state lock table"
  }
}

# SQS — Ansible Job Queue + Dead Letter Queue
resource "aws_sqs_queue" "ansible_jobs_dlq" {
  name                      = "ansible-engine-jobs-dlq"
  message_retention_seconds = 1209600 # 14 days
}

resource "aws_sqs_queue" "ansible_jobs" {
  name                       = "ansible-engine-jobs-queue"
  visibility_timeout_seconds = 900    # 15 minutes
  message_retention_seconds  = 345600 # 4 days
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.ansible_jobs_dlq.arn
    maxReceiveCount     = 3
  })
}



data "aws_caller_identity" "current" {}

# Subnet lookup for Fargate task networking
data "aws_subnets" "default" {
  filter {
    name   = "default-for-az"
    values = ["true"]
  }
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
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["*"]
  zip_dir_slice = "authorizer"
  environment_variables = {
    JWT_SECRET = var.jwt_secret
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
    PSQL_CONNECTION_STRING      = var.psql_connection_string
    CLOUDFORMATION_TEMPLATE_URL = var.cloudformation_template_url
    CLUTTER_ACCOUNT_ID          = var.clutter_account_id
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
  }

}
# Organization Accounts
module "organization-submit-account-role-arn-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-submit-account-role-arn"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-submit-account-role-arn:*"
  ]
  zip_dir_slice = "organization/accounts/submit-role-arn"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "organization-get-accounts-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-get-accounts"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-get-accounts:*"
  ]
  zip_dir_slice = "organization/accounts/get"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}
module "organization-delete-account-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "organization-delete-account"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/organization-delete-account:*"
  ]
  zip_dir_slice = "organization/accounts/delete"
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
    CORS_ALLOWED_ORIGIN    = var.frontend_url
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
  resources             = ["arn:aws:logs:*:*:log-group:/aws/lambda/user-information-get:*"]
  zip_dir_slice         = "user-information/get"
  environment_variables = {}

}

# Terraform Engine
module "terraform-engine-create-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-engine-create"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:PutObject",
    "s3:GetObject"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-engine-create:*",
    "arn:aws:s3:::${var.terraform_output_bucket}/*",
    "arn:aws:s3:::${var.terraform_template_bucket}/*"
  ]
  zip_dir_slice = "terraform-engine/create"
  environment_variables = {
    S3_BUCKET_NAME         = var.terraform_output_bucket
    TEMPLATE_BUCKET_NAME   = var.terraform_template_bucket
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

# ==================================
# Ansible Engine Lambda Functions
# ==================================

module "ansible-submit-job-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-submit-job"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",

    "sqs:SendMessage"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-submit-job:*",

    aws_sqs_queue.ansible_jobs.arn
  ]
  zip_dir_slice = "ansible/submit-job"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    JOB_QUEUE_URL          = aws_sqs_queue.ansible_jobs.url
    CORS_ALLOWED_ORIGIN    = var.frontend_url
  }
}

module "ansible-get-job-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-get-job"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",

  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-get-job:*",

  ]
  zip_dir_slice = "ansible/get-job"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    CORS_ALLOWED_ORIGIN    = var.frontend_url
  }
}

module "ansible-list-jobs-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-list-jobs"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",

  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-list-jobs:*",

  ]
  zip_dir_slice = "ansible/list-jobs"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    CORS_ALLOWED_ORIGIN    = var.frontend_url
  }
}

module "ansible-get-job-logs-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-get-job-logs"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:GetObject",
    "s3:ListBucket",
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-get-job-logs:*",
    module.s3.clutter_bucket_arn,
    "${module.s3.clutter_bucket_arn}/logs/*",
  ]
  zip_dir_slice = "ansible/get-job-logs"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN    = var.frontend_url
  }
}

module "ansible-create-playbook-upload-url-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-create-playbook-upload-url"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:PutObject"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-create-playbook-upload-url:*",
    "${module.s3.clutter_bucket_arn}/*"
  ]
  zip_dir_slice = "ansible/create-playbook-upload-url"
  environment_variables = {
    S3_BUCKET_NAME      = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN = var.frontend_url
  }
}

module "ansible-run-task-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-run-task"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "ecs:RunTask",
    "iam:PassRole",

  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-run-task:*",
    module.fargate.cluster_arn,
    "arn:aws:ecs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:task-definition/ansible-runner:*",

    module.fargate.ansible_task_role_arn,
    module.fargate.ecs_execution_role_arn
  ]
  zip_dir_slice = "ansible/run-task"
  environment_variables = {
    PSQL_CONNECTION_STRING      = var.psql_connection_string
    ANSIBLE_ECS_CLUSTER_ARN     = module.fargate.cluster_arn
    ANSIBLE_TASK_DEFINITION_ARN = module.fargate.ansible_task_def_arn
    ANSIBLE_SUBNET_IDS          = join(",", data.aws_subnets.default.ids)
    ANSIBLE_SECURITY_GROUP_ID   = module.fargate.ansible_sg_id
    S3_BUCKET_NAME              = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN         = var.frontend_url
  }
}

# SQS → Lambda trigger for ansible-run-task
resource "aws_lambda_event_source_mapping" "ansible_sqs_trigger" {
  event_source_arn = aws_sqs_queue.ansible_jobs.arn
  function_name    = module.ansible-run-task-lambda.arn
  batch_size       = 1
  enabled          = true
}

# Grant SQS permissions to the run-task lambda
resource "aws_iam_role_policy" "ansible_run_task_sqs" {
  name = "ansible-run-task-sqs-policy"
  role = module.ansible-run-task-lambda.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "sqs:ReceiveMessage",
        "sqs:DeleteMessage",
        "sqs:GetQueueAttributes"
      ]
      Resource = aws_sqs_queue.ansible_jobs.arn
    }]
  })
}

# ===========
# API Gateway
# ===========

module "clutter-api-gateway" {
  source             = "./modules/api-gateway"
  aws_region         = var.aws_region
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
# Organization Accounts
module "organization-accounts-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.organization-api-path-by-id.resource_id
  path_part   = "accounts"
}
module "organization-accounts-api-path-by-account-id" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.organization-accounts-api-path.resource_id
  path_part   = "{accountId}"
}
module "organization-accounts-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.organization-accounts-api-path.resource_id
  http_methods = ["GET"]
}
module "organization-accounts-api-by-account-id-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.organization-accounts-api-path-by-account-id.resource_id
  http_methods = ["POST", "DELETE"]
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

# Terraform Engine
module "terraform-engine-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "terraform-engine"
}
module "terraform-engine-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-engine-api-path.resource_id
  http_methods = ["POST", "GET"]
}

# Ansible Engine
module "ansible-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "ansible"
}
module "ansible-jobs-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.ansible-api-path.resource_id
  path_part   = "jobs"
}
module "ansible-jobs-by-id-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.ansible-jobs-api-path.resource_id
  path_part   = "{jobId}"
}
module "ansible-jobs-by-id-logs-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.ansible-jobs-by-id-api-path.resource_id
  path_part   = "logs"
}
module "ansible-playbooks-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.ansible-api-path.resource_id
  path_part   = "playbooks"
}
module "ansible-playbooks-upload-url-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.ansible-playbooks-api-path.resource_id
  path_part   = "upload-url"
}
module "ansible-jobs-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.ansible-jobs-api-path.resource_id
  http_methods = ["POST", "GET"]
}
module "ansible-jobs-by-id-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.ansible-jobs-by-id-api-path.resource_id
  http_methods = ["GET"]
}
module "ansible-jobs-by-id-logs-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.ansible-jobs-by-id-logs-api-path.resource_id
  http_methods = ["GET"]
}
module "ansible-playbooks-upload-url-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.ansible-playbooks-upload-url-api-path.resource_id
  http_methods = ["POST"]
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
module "organization-submit-account-role-arn-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "organizationSubmitAccountRoleArn"
  description     = "Model to validate organization account role arn submission requests"
  schema_filename = "organization-submit-account-role-arn.json"
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
module "ansible-playbook-upload-create-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "ansiblePlaybookUploadCreate"
  description     = "Model to validate ansible playbook upload URL requests"
  schema_filename = "ansible-playbook-upload-create.json"
}
module "ansible-job-submit-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "ansibleJobSubmit"
  description     = "Model to validate ansible job submission requests"
  schema_filename = "ansible-job-submit.json"
}

# Integrations

# POST Log-in
module "log-in-api-integration" {
  source        = "./modules/templates/api-lambda-integration"
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  resource_id   = module.log-in-api-path.resource_id
  http_method   = "POST"
  invoke_arn    = module.log-in-lambda.invoke_arn
  function_name = module.log-in-lambda.function_name
  path_part     = module.log-in-api-path.path_part
  execution_arn = module.clutter-api-gateway.execution_arn
  path          = module.log-in-api-path.path

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
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-api-path.resource_id
  http_method       = "POST"
  invoke_arn        = module.organization-create-lambda.invoke_arn
  function_name     = module.organization-create-lambda.function_name
  path_part         = module.organization-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

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
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# UPDATE organization
module "organization-update-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-api-path-by-id.resource_id
  http_method       = "PUT"
  invoke_arn        = module.organization-update-lambda.invoke_arn
  function_name     = module.organization-update-lambda.function_name
  path_part         = module.organization-api-path-by-id.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-api-path-by-id.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

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

# Organization Accounts
# GET organization/{organizationId}/accounts
module "organization-get-accounts-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-accounts-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.organization-get-accounts-lambda.invoke_arn
  function_name     = module.organization-get-accounts-lambda.function_name
  path_part         = module.organization-accounts-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-accounts-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

# POST organization/{organizationId}/accounts/{accountId}
module "organization-submit-account-role-arn-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-accounts-api-path-by-account-id.resource_id
  http_method       = "POST"
  invoke_arn        = module.organization-submit-account-role-arn-lambda.invoke_arn
  function_name     = module.organization-submit-account-role-arn-lambda.function_name
  path_part         = module.organization-accounts-api-path-by-account-id.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-accounts-api-path-by-account-id.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.organization-submit-account-role-arn-model.model_name
}

# DELETE organization/{organizationId}/accounts
module "organization-delete-account-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.organization-accounts-api-path-by-account-id.resource_id
  http_method       = "DELETE"
  invoke_arn        = module.organization-delete-account-lambda.invoke_arn
  function_name     = module.organization-delete-account-lambda.function_name
  path_part         = module.organization-accounts-api-path-by-account-id.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.organization-accounts-api-path-by-account-id.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

# Project
# POST project
module "project-create-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.project-api-path.resource_id
  http_method       = "POST"
  invoke_arn        = module.project-create-lambda.invoke_arn
  function_name     = module.project-create-lambda.function_name
  path_part         = module.project-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.project-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

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
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.project-api-path.resource_id
  http_method       = "PUT"
  invoke_arn        = module.project-update-lambda.invoke_arn
  function_name     = module.project-update-lambda.function_name
  path_part         = module.project-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.project-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

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

# Ansible Engine
# POST ansible/jobs
module "ansible-submit-job-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.ansible-jobs-api-path.resource_id
  http_method       = "POST"
  invoke_arn        = module.ansible-submit-job-lambda.invoke_arn
  function_name     = module.ansible-submit-job-lambda.function_name
  path_part         = module.ansible-jobs-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.ansible-jobs-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.ansible-job-submit-model.model_name
}
# GET ansible/jobs
module "ansible-list-jobs-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.ansible-jobs-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.ansible-list-jobs-lambda.invoke_arn
  function_name     = module.ansible-list-jobs-lambda.function_name
  path_part         = module.ansible-jobs-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.ansible-jobs-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# GET ansible/jobs/{jobId}
module "ansible-get-job-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.ansible-jobs-by-id-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.ansible-get-job-lambda.invoke_arn
  function_name     = module.ansible-get-job-lambda.function_name
  path_part         = module.ansible-jobs-by-id-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.ansible-jobs-by-id-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# GET ansible/jobs/{jobId}/logs
module "ansible-get-job-logs-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.ansible-jobs-by-id-logs-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.ansible-get-job-logs-lambda.invoke_arn
  function_name     = module.ansible-get-job-logs-lambda.function_name
  path_part         = module.ansible-jobs-by-id-logs-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.ansible-jobs-by-id-logs-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}
# POST ansible/playbooks/upload-url
module "ansible-create-playbook-upload-url-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.ansible-playbooks-upload-url-api-path.resource_id
  http_method       = "POST"
  invoke_arn        = module.ansible-create-playbook-upload-url-lambda.invoke_arn
  function_name     = module.ansible-create-playbook-upload-url-lambda.function_name
  path_part         = module.ansible-playbooks-upload-url-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.ansible-playbooks-upload-url-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id

  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.ansible-playbook-upload-create-model.model_name
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

# Terraform Engine
# POST terraform-engine
module "terraform-engine-create-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-engine-api-path.resource_id
  http_method       = "POST"
  invoke_arn        = module.terraform-engine-create-lambda.invoke_arn
  function_name     = module.terraform-engine-create-lambda.function_name
  path_part         = module.terraform-engine-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-engine-api-path.path
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
      module.organization-submit-account-role-arn-model.model_id,

      module.log-in-api-integration.integration_id,

      module.cloudformation-stack-url-generator-api-integration.integration_id,

      module.organization-create-api-integration.integration_id,
      module.organization-get-api-integration.integration_id,
      module.organization-update-api-integration.integration_id,
      module.organization-delete-api-integration.integration_id,

      module.organization-submit-account-role-arn-api-integration.integration_id,
      module.organization-get-accounts-api-integration.integration_id,
      module.organization-delete-account-api-integration.integration_id,

      module.project-create-api-integration.integration_id,
      module.project-get-api-integration.integration_id,
      module.project-update-api-integration.integration_id,
      module.project-delete-api-integration.integration_id,

      module.diagram-create-api-integration.integration_id,
      module.diagram-get-api-integration.integration_id,
      module.diagram-update-api-integration.integration_id,
      module.diagram-delete-api-integration.integration_id,

      module.user-information-get-api-integration.integration_id,

      module.terraform-engine-create-api-integration.integration_id,
      module.ansible-submit-job-api-integration.integration_id,
      module.ansible-list-jobs-api-integration.integration_id,
      module.ansible-get-job-api-integration.integration_id,
      module.ansible-get-job-logs-api-integration.integration_id,
      module.ansible-create-playbook-upload-url-api-integration.integration_id,
      module.ansible-playbook-upload-create-model.model_id,
      module.ansible-job-submit-model.model_id
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

variable "ansible_runner_image_tag" {
  type        = string
  description = "Docker image tag for the Ansible runner"
  default     = "latest"
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
