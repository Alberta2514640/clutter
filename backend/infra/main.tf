# ===
# S3
# ===

module "s3" {
  source = "./modules/s3"

  aws_region             = var.aws_region
  enable_versioning      = true
  enable_cors            = true
  cors_allowed_origins   = ["*"]
  ansible_runner_role_id = module.fargate.ansible_task_role_unique_id
}

# ================
# ECS Fargate Task
# ================

module "fargate" {
  source = "./modules/fargate"

  s3_clutter_arn    = module.s3.clutter_bucket_arn
  s3_clutter_name   = module.s3.clutter_bucket_name
  s3_templates_arn  = module.s3.clutter_templates_bucket_arn
  s3_templates_name = module.s3.clutter_templates_bucket_name

  aws_region                   = var.aws_region
  ansible_runner_image_tag     = var.ansible_runner_image_tag
  terraform_deployer_image_tag = var.terraform_deployer_image_tag
}

# The site's origin, from its one name: A's Lambdas and B answer CORS for it
# alone. B is served on the same origin (paths /v1/* and /healthz).
locals {
  frontend_url = "https://${var.site_domain}"
}

# =========================================
# B: the adapter layer's API and run tasks
# =========================================
# Its own VPC, cluster, roles and buckets, and the load balancer the website
# shares (docs/architecture/environments.md).

module "adapter-service" {
  source = "./modules/adapter-service"

  aws_region            = var.aws_region
  image_tag             = var.adapter_image_tag
  certificate_arn       = var.certificate_arn
  domain                = var.site_domain
  frontend_url          = local.frontend_url
  database_url          = var.psql_connection_string
  jwt_secret            = var.jwt_secret
  templates_bucket_name = module.s3.clutter_templates_bucket_name
  templates_bucket_arn  = module.s3.clutter_templates_bucket_arn
}

output "load_balancer_dns_name" {
  description = "Point site_domain at this with a CNAME or alias record"
  value       = module.adapter-service.load_balancer_dns_name
}

output "adapter_api_repository_url" {
  value = module.adapter-service.api_repository_url
}

output "adapter_runtask_repository_url" {
  value = module.adapter-service.runtask_repository_url
}

# ==========================================
# The website, behind B's load balancer
# ==========================================
# Its own cluster, image repository, role and security group; the load
# balancer sends site_domain to it, except B's paths (docs/architecture/environments.md).

module "website" {
  source = "./modules/website"

  aws_region            = var.aws_region
  image_tag             = var.site_image_tag
  domain                = var.site_domain
  vpc_id                = module.adapter-service.vpc_id
  subnet_ids            = module.adapter-service.public_subnet_ids
  alb_security_group_id = module.adapter-service.alb_security_group_id
  listener_arn          = module.adapter-service.https_listener_arn
}

output "website_repository_url" {
  value = module.website.repository_url
}

# ========================
# Ansible Engine Resources
# ========================

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
    GOOGLE_CLIENT_ID       = var.google_client_id
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
    PSQL_CONNECTION_STRING = var.psql_connection_string
    # The key is the one modules/s3 uploads the role template to; the bucket is public-read, which CloudFormation needs.
    CLOUDFORMATION_TEMPLATE_URL = "https://${module.s3.clutter_templates_bucket_name}.s3.${var.aws_region}.amazonaws.com/templates/cloudformation/client_side_terraform_deployer_role.yaml"
    CLUTTER_ACCOUNT_ID          = var.clutter_account_id
  }
}

module "terraform-command-runner-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-command-runner"

  actions = [
    "ecs:RunTask",
    "iam:PassRole"
  ]

  resources = [
    module.fargate.terraform_task_role_arn,
    module.fargate.ecs_execution_role_arn,
    module.fargate.terraform_task_definition_arn
  ]

  zip_dir_slice = "terraform-command-runner"

  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    ECS_CLUSTER_NAME       = module.fargate.ecs_cluster_name
    TASK_DEFINITION_ARN    = module.fargate.terraform_task_definition_arn
    CONTAINER_NAME         = module.fargate.terraform_container_name
    SUBNET_IDS             = join(",", module.fargate.subnet_ids)
    SECURITY_GROUP_ID      = module.fargate.terraform_security_group_id
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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

# Resources
module "resources-get-lambda" {

  source        = "./modules/templates/lambda"
  function_name = "resources-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources     = ["arn:aws:logs:*:*:log-group:/aws/lambda/resources-get:*"]
  zip_dir_slice = "resources/get"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }

}

# Terraform Engine
module "terraform-engine-code-upload-presigned-url-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-engine-code-upload-presigned-url"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:PutObject"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-engine-code-upload-presigned-url:*",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}/*/terraform/code/*/bootstrap.zip"
  ]
  zip_dir_slice = "terraform-engine/code-upload"
  environment_variables = {
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

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
    "arn:aws:s3:::${module.s3.clutter_bucket_name}/*",
    "arn:aws:s3:::${module.s3.clutter_templates_bucket_name}/*"
  ]
  zip_dir_slice = "terraform-engine/create"
  environment_variables = {
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    TEMPLATE_BUCKET_NAME   = module.s3.clutter_templates_bucket_name
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

# A run deletes a generated file only when it has no content, and only
# outputs.tf can be empty. Nothing else in the folder may be deleted, above all
# the runner's state beside it.
resource "aws_iam_role_policy" "terraform_engine_create_delete_outputs" {
  name = "terraform-engine-create-delete-outputs-policy"
  role = module.terraform-engine-create-lambda.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "s3:DeleteObject"
      Resource = "arn:aws:s3:::${module.s3.clutter_bucket_name}/*/terraform/outputs.tf"
    }]
  })
}

module "terraform-engine-files-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-engine-files"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-engine-files:*"
  ]
  zip_dir_slice = "terraform-engine/files"
  # One database query and up to a few S3 reads, on a cold start.
  timeout = 10
  environment_variables = {
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

# The folder also holds the runner's state and deploy logs, so reads stop at the
# generated files and the status. The state file's name does not end in .tf.
# ListBucket stays on the bucket: without it S3 answers a missing key with 403,
# not 404, and the route could not tell a file deleted mid-read from a failure.
resource "aws_iam_role_policy" "terraform_engine_files_read" {
  name = "terraform-engine-files-read-policy"
  role = module.terraform-engine-files-lambda.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:ListBucket"
        Resource = "arn:aws:s3:::${module.s3.clutter_bucket_name}"
      },
      {
        Effect = "Allow"
        Action = "s3:GetObject"
        Resource = [
          "arn:aws:s3:::${module.s3.clutter_bucket_name}/*/terraform/*.tf",
          "arn:aws:s3:::${module.s3.clutter_bucket_name}/*/terraform/status.json"
        ]
      }
    ]
  })
}

# ==================================
# Deployment logs Lambda Functions
# ==================================

# TODO:
# All log functions
# ie. terraform-engine-logs-live-lambda, terraform-engine-logs-url-lambda, terraform-engine-logs-get-lambda
# should have their prefix, namely "terraform-engine", changed to "terraform-command-runner"
# for correctness

module "terraform-engine-logs-get-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-engine-logs-get"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:ListBucket",
    "s3:GetObject"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-engine-logs-get:*",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}/*"
  ]
  zip_dir_slice = "terraform-engine/logs/get"
  environment_variables = {
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

module "terraform-engine-logs-url-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-engine-logs-url"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:GetObject"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-engine-logs-url:*",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}/*"
  ]
  zip_dir_slice = "terraform-engine/logs/url"
  environment_variables = {
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

module "terraform-engine-logs-live-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-engine-logs-live"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "ecs:DescribeTasks",
    "logs:GetLogEvents",
    "s3:GetObject",
    "s3:ListBucket"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-engine-logs-live:*",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}/*",
    module.fargate.ecs_cluster_arn,
    "arn:aws:ecs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:task/${module.fargate.ecs_cluster_name}/*",
    "arn:aws:logs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:log-group:/ecs/terraform-deployer:*"
  ]
  zip_dir_slice = "terraform-engine/logs/live"
  environment_variables = {
    PSQL_CONNECTION_STRING               = var.psql_connection_string
    ECS_CLUSTER_NAME                     = module.fargate.ecs_cluster_name
    TERRAFORM_DEPLOYER_LOG_GROUP         = "/ecs/terraform-deployer"
    TERRAFORM_DEPLOYER_LOG_STREAM_PREFIX = "ecs"
    TERRAFORM_DEPLOYER_CONTAINER_NAME    = "terraform-deployer"
    S3_BUCKET_NAME                       = module.s3.clutter_bucket_name
  }
}

# The prefix for this Lambda has already been fixed to "terraform-command-runner"

# Terraform Command Runner logs to populate "Recent Activity" table in frontend
module "terraform-command-runner-logs-recent-activity-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "terraform-command-runner-logs-recent-activity"
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:GetObject",
    "s3:ListBucket"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/terraform-command-runner-logs-recent-activity:*",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}",
    "arn:aws:s3:::${module.s3.clutter_bucket_name}/*"
  ]
  zip_dir_slice = "terraform-engine/logs/recent-activity"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
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
    "sqs:SendMessage",
    "s3:GetObject"
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-submit-job:*",
    aws_sqs_queue.ansible_jobs.arn,
    "${module.s3.clutter_bucket_arn}/*"
  ]
  zip_dir_slice = "ansible/submit-job"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    JOB_QUEUE_URL          = aws_sqs_queue.ansible_jobs.url
    CORS_ALLOWED_ORIGIN    = local.frontend_url
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
  }
}

module "ansible-get-job-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-get-job"
  # S3 reads the runner's job status file (generic/job_status.go); ListBucket
  # makes a missing file a 404 rather than a 403.
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:GetObject",
    "s3:ListBucket",
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-get-job:*",
    module.s3.clutter_bucket_arn,
    "${module.s3.clutter_bucket_arn}/*",
  ]
  zip_dir_slice = "ansible/get-job"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN    = local.frontend_url
  }
}

module "ansible-list-jobs-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-list-jobs"
  # S3 reads the runner's job status files (generic/job_status.go); ListBucket
  # makes a missing file a 404 rather than a 403.
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "s3:GetObject",
    "s3:ListBucket",
  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-list-jobs:*",
    module.s3.clutter_bucket_arn,
    "${module.s3.clutter_bucket_arn}/*",
  ]
  zip_dir_slice = "ansible/list-jobs"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    "${module.s3.clutter_bucket_arn}/*",
  ]
  zip_dir_slice = "ansible/get-job-logs"
  environment_variables = {
    PSQL_CONNECTION_STRING = var.psql_connection_string
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN    = local.frontend_url
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
    S3_BUCKET_NAME         = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN    = local.frontend_url
    PSQL_CONNECTION_STRING = var.psql_connection_string
  }
}

module "ansible-run-task-lambda" {
  source        = "./modules/templates/lambda"
  function_name = "ansible-run-task"
  timeout       = 480
  actions = [
    "logs:CreateLogGroup",
    "logs:CreateLogStream",
    "logs:PutLogEvents",
    "ecs:RunTask",
    "iam:PassRole",

  ]
  resources = [
    "arn:aws:logs:*:*:log-group:/aws/lambda/ansible-run-task:*",
    module.fargate.ecs_cluster_arn,
    "arn:aws:ecs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:task-definition/ansible-runner:*",

    module.fargate.ansible_task_role_arn,
    module.fargate.ecs_execution_role_arn
  ]
  zip_dir_slice = "ansible/run-task"
  environment_variables = {
    PSQL_CONNECTION_STRING      = var.psql_connection_string
    ANSIBLE_ECS_CLUSTER_ARN     = module.fargate.ecs_cluster_arn
    ANSIBLE_TASK_DEFINITION_ARN = module.fargate.ansible_task_def_arn
    ANSIBLE_SUBNET_IDS          = join(",", data.aws_subnets.default.ids)
    ANSIBLE_SECURITY_GROUP_ID   = module.fargate.ansible_sg_id
    S3_BUCKET_NAME              = module.s3.clutter_bucket_name
    CORS_ALLOWED_ORIGIN         = local.frontend_url
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

# Grant EC2 start permissions to the run-task lambda
resource "aws_iam_role_policy" "ansible_run_task_ec2_start" {
  name = "ansible-run-task-ec2-start-policy"
  role = module.ansible-run-task-lambda.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "ec2:StartInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus"
      ]
      Resource = "*"
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

# Teraform Command
module "terraform-command-runner-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "terraform-command-runner"
}
module "terraform-command-runner-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-command-runner-api-path.resource_id
  http_methods = ["POST"]
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

# Resources
module "resources-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.clutter-api-gateway.root_resource_id
  path_part   = "resources"
}
module "resources-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.resources-api-path.resource_id
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

# Generated Terraform files
module "terraform-engine-files-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-api-path.resource_id
  path_part   = "files"
}

module "terraform-engine-files-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-engine-files-api-path.resource_id
  http_methods = ["GET"]
}

# Terraform logs
module "terraform-engine-logs-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-api-path.resource_id
  path_part   = "logs"
}

module "terraform-engine-logs-url-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-logs-api-path.resource_id
  path_part   = "url"
}

module "terraform-engine-logs-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-engine-logs-api-path.resource_id
  http_methods = ["GET"]
}

module "terraform-engine-logs-url-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-engine-logs-url-api-path.resource_id
  http_methods = ["GET"]
}

module "terraform-engine-logs-live-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-logs-api-path.resource_id
  path_part   = "live"
}

module "terraform-engine-logs-live-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-engine-logs-live-api-path.resource_id
  http_methods = ["GET"]
}

module "terraform-command-runner-logs-recent-activity-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-logs-api-path.resource_id
  path_part   = "recent-activity"
}

module "terraform-command-runner-logs-recent-activity-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-command-runner-logs-recent-activity-api-path.resource_id
  http_methods = ["GET"]
}

# Terraform Engine Code Upload
module "terraform-engine-code-upload-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-api-path.resource_id
  path_part   = "code-upload"
}

module "terraform-engine-code-upload-presigned-url-api-path" {
  source      = "./modules/templates/api-path"
  rest_api_id = module.clutter-api-gateway.rest_api_id
  parent_id   = module.terraform-engine-code-upload-api-path.resource_id
  path_part   = "presigned-url"
}

module "terraform-engine-code-upload-presigned-url-api-cors-compliance" {
  source       = "./modules/templates/api-path-cors-compliance"
  rest_api_id  = module.clutter-api-gateway.rest_api_id
  resource_id  = module.terraform-engine-code-upload-presigned-url-api-path.resource_id
  http_methods = ["POST"]
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

# Terraform Command
module "terraform-command-runner-model" {
  source          = "./modules/templates/api-models"
  rest_api_id     = module.clutter-api-gateway.rest_api_id
  model_name      = "terraformCommand"
  description     = "Model to validate terraform command requests"
  schema_filename = "terraform-command-runner.json"
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

# Terraform Command
# POST terraform-command-runner
module "terraform-command-runner-api-integration" {
  source               = "./modules/templates/api-lambda-integration"
  rest_api_id          = module.clutter-api-gateway.rest_api_id
  resource_id          = module.terraform-command-runner-api-path.resource_id
  http_method          = "POST"
  invoke_arn           = module.terraform-command-runner-lambda.invoke_arn
  function_name        = module.terraform-command-runner-lambda.function_name
  path_part            = module.terraform-command-runner-api-path.path_part
  execution_arn        = module.clutter-api-gateway.execution_arn
  path                 = module.terraform-command-runner-api-path.path
  request_validator_id = module.clutter-api-gateway.body_validator_id
  model_name           = module.terraform-command-runner-model.model_name
  jwt_authorizer_id    = module.clutter-api-gateway.jwt_authorizer_id
}

module "terraform-engine-files-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-engine-files-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.terraform-engine-files-lambda.invoke_arn
  function_name     = module.terraform-engine-files-lambda.function_name
  path_part         = module.terraform-engine-files-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-engine-files-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

module "terraform-engine-logs-get-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-engine-logs-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.terraform-engine-logs-get-lambda.invoke_arn
  function_name     = module.terraform-engine-logs-get-lambda.function_name
  path_part         = module.terraform-engine-logs-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-engine-logs-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

module "terraform-engine-logs-url-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-engine-logs-url-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.terraform-engine-logs-url-lambda.invoke_arn
  function_name     = module.terraform-engine-logs-url-lambda.function_name
  path_part         = module.terraform-engine-logs-url-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-engine-logs-url-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

module "terraform-engine-logs-live-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-engine-logs-live-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.terraform-engine-logs-live-lambda.invoke_arn
  function_name     = module.terraform-engine-logs-live-lambda.function_name
  path_part         = module.terraform-engine-logs-live-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-engine-logs-live-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

module "terraform-command-runner-logs-recent-activity-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-command-runner-logs-recent-activity-api-path.resource_id
  http_method       = "GET"
  invoke_arn        = module.terraform-command-runner-logs-recent-activity-lambda.invoke_arn
  function_name     = module.terraform-command-runner-logs-recent-activity-lambda.function_name
  path_part         = module.terraform-command-runner-logs-recent-activity-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-command-runner-logs-recent-activity-api-path.path
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

# Resources
# GET resources
# Public on purpose: it has no jwt_authorizer_id because it serves only the
# static resource catalog (the supported_resources seed), which holds no
# tenant data. Every other route except POST /log-in has the authorizer.
module "resources-get-api-integration" {
  source        = "./modules/templates/api-lambda-integration"
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  resource_id   = module.resources-api-path.resource_id
  http_method   = "GET"
  invoke_arn    = module.resources-get-lambda.invoke_arn
  function_name = module.resources-get-lambda.function_name
  path_part     = module.resources-api-path.path_part
  execution_arn = module.clutter-api-gateway.execution_arn
  path          = module.resources-api-path.path
}

# Terraform Engine
# POST terraform-engine: the editor calls it with the user's token after a save
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
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

module "terraform-engine-code-upload-presigned-url-api-integration" {
  source            = "./modules/templates/api-lambda-integration"
  rest_api_id       = module.clutter-api-gateway.rest_api_id
  resource_id       = module.terraform-engine-code-upload-presigned-url-api-path.resource_id
  http_method       = "POST"
  invoke_arn        = module.terraform-engine-code-upload-presigned-url-lambda.invoke_arn
  function_name     = module.terraform-engine-code-upload-presigned-url-lambda.function_name
  path_part         = module.terraform-engine-code-upload-presigned-url-api-path.path_part
  execution_arn     = module.clutter-api-gateway.execution_arn
  path              = module.terraform-engine-code-upload-presigned-url-api-path.path
  jwt_authorizer_id = module.clutter-api-gateway.jwt_authorizer_id
}

# ======================
# API Gateway: the stage
# ======================
# A REST API answers only through a stage, and a stage serves a deployment: a
# snapshot of the routes, methods and integrations taken when it is created.
# The trigger hashes every file that defines the API (this file, the
# api-gateway module, and the api-* templates with their model schemas), so a
# change to any route, method, integration, model or the authorizer makes a new
# deployment on the next apply. An edit elsewhere in this file also makes one;
# that is harmless, the stage moves to an identical snapshot.

locals {
  api_definition_files = sort(concat(
    ["main.tf"],
    tolist(fileset(path.module, "modules/api-gateway/*.tf")),
    tolist(fileset(path.module, "modules/templates/api-*/**")),
  ))
}

# Errors API Gateway answers itself (a token the authorizer refuses, a missing
# route, a throttle, a timeout) carry the CORS origin header too. Without it the
# browser hides the 403 as a CORS failure, so the website cannot tell an expired
# session and never sends the user back to sign in. The devserver's gateway
# mirrors this (backend/devserver/internal/gateway/server.go, writeGatewayError).
resource "aws_api_gateway_gateway_response" "default-4xx" {
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  response_type = "DEFAULT_4XX"
  response_parameters = {
    "gatewayresponse.header.Access-Control-Allow-Origin" = "'*'"
  }
}

resource "aws_api_gateway_gateway_response" "default-5xx" {
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  response_type = "DEFAULT_5XX"
  response_parameters = {
    "gatewayresponse.header.Access-Control-Allow-Origin" = "'*'"
  }
}

resource "aws_api_gateway_deployment" "clutter-api" {
  rest_api_id = module.clutter-api-gateway.rest_api_id

  triggers = {
    redeployment = sha1(join(",", [for f in local.api_definition_files : filesha1("${path.module}/${f}")]))
  }

  lifecycle {
    create_before_destroy = true
  }

  # A deployment fails while any method lacks its integration, so it waits for
  # every route. A new route's integration and CORS modules go in this list;
  # backend/infra/test-deploy.sh fails until they do.
  depends_on = [
    module.log-in-api-cors-compliance,
    module.cloudformation-stack-url-generator-api-by-id-cors-compliance,
    module.terraform-command-runner-api-cors-compliance,
    module.organization-api-cors-compliance,
    module.organization-api-by-id-cors-compliance,
    module.organization-accounts-api-cors-compliance,
    module.organization-accounts-api-by-account-id-cors-compliance,
    module.project-api-cors-compliance,
    module.diagram-api-cors-compliance,
    module.user-information-api-cors-compliance,
    module.resources-api-cors-compliance,
    module.terraform-engine-api-cors-compliance,
    module.terraform-engine-files-api-cors-compliance,
    module.terraform-engine-logs-api-cors-compliance,
    module.terraform-engine-logs-url-api-cors-compliance,
    module.terraform-engine-logs-live-api-cors-compliance,
    module.terraform-command-runner-logs-recent-activity-api-cors-compliance,
    module.terraform-engine-code-upload-presigned-url-api-cors-compliance,
    module.ansible-jobs-api-cors-compliance,
    module.ansible-jobs-by-id-api-cors-compliance,
    module.ansible-jobs-by-id-logs-api-cors-compliance,
    module.ansible-playbooks-upload-url-api-cors-compliance,
    module.log-in-api-integration,
    module.cloudformation-stack-url-generator-api-integration,
    module.organization-create-api-integration,
    module.organization-get-api-integration,
    module.organization-update-api-integration,
    module.organization-delete-api-integration,
    module.organization-get-accounts-api-integration,
    module.organization-submit-account-role-arn-api-integration,
    module.organization-delete-account-api-integration,
    module.project-create-api-integration,
    module.project-get-api-integration,
    module.project-update-api-integration,
    module.project-delete-api-integration,
    module.diagram-create-api-integration,
    module.diagram-get-api-integration,
    module.diagram-update-api-integration,
    module.diagram-delete-api-integration,
    module.terraform-command-runner-api-integration,
    module.terraform-engine-files-api-integration,
    module.terraform-engine-logs-get-api-integration,
    module.terraform-engine-logs-url-api-integration,
    module.terraform-engine-logs-live-api-integration,
    module.terraform-command-runner-logs-recent-activity-api-integration,
    module.ansible-submit-job-api-integration,
    module.ansible-list-jobs-api-integration,
    module.ansible-get-job-api-integration,
    module.ansible-get-job-logs-api-integration,
    module.ansible-create-playbook-upload-url-api-integration,
    module.user-information-get-api-integration,
    module.resources-get-api-integration,
    module.terraform-engine-create-api-integration,
    module.terraform-engine-code-upload-presigned-url-api-integration,
    aws_api_gateway_gateway_response.default-4xx,
    aws_api_gateway_gateway_response.default-5xx,
  ]
}

resource "aws_api_gateway_stage" "prod" {
  rest_api_id   = module.clutter-api-gateway.rest_api_id
  deployment_id = aws_api_gateway_deployment.clutter-api.id
  stage_name    = "prod"
}

output "api_invoke_url" {
  description = "A's API address; npm run deploy -- prod --site-image builds the website with it as NEXT_PUBLIC_API_ENDPOINT"
  value       = aws_api_gateway_stage.prod.invoke_url
}
