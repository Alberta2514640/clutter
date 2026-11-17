# B's four roles (docs/architecture/system-b-adapter-layer.md § Isolating the runner):
#
#   ClutterAdapterRunnerRole        B API's task role. Users' AWS roles trust it by name
#                                   (AdapterRunnerRoleName in client_side_terraform_deployer_role.yaml).
#   ClutterAdapterRunFilesRole      assumed by B API once per run, narrowed by a session
#                                   policy to that run's objects and that deployment's state;
#                                   its keys are the only credentials a run task holds.
#   ClutterAdapterApiExecutionRole  lets ECS pull B API's image and read its two secrets.
#   ClutterAdapterRunExecutionRole  lets ECS pull the run task's image and read a run's task.env.
#
# The run task has no task role: no database, no secret store, no AWS access of its own.

locals {
  account_id      = data.aws_caller_identity.current.account_id
  run_family      = "clutter-adapter-run"
  connection_arns = "arn:aws:secretsmanager:${var.aws_region}:${local.account_id}:secret:${var.secrets_prefix}*"
}

data "aws_iam_policy_document" "ecs_tasks_trust" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [local.account_id]
    }
  }
}

# --- B API's task role ---

resource "aws_iam_role" "runner" {
  name               = var.runner_role_name
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
}

resource "aws_iam_role_policy" "runner" {
  name = "adapter-api"
  role = aws_iam_role.runner.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ConnectionSecrets"
        Effect   = "Allow"
        Action   = ["secretsmanager:CreateSecret", "secretsmanager:GetSecretValue", "secretsmanager:DeleteSecret"]
        Resource = local.connection_arns
      },
      {
        Sid       = "StartRunTasks"
        Effect    = "Allow"
        Action    = "ecs:RunTask"
        Resource  = "arn:aws:ecs:${var.aws_region}:${local.account_id}:task-definition/${local.run_family}:*"
        Condition = { ArnEquals = { "ecs:cluster" = aws_ecs_cluster.adapter.arn } }
      },
      {
        Sid      = "FollowRunTasks"
        Effect   = "Allow"
        Action   = ["ecs:DescribeTasks", "ecs:StopTask"]
        Resource = "arn:aws:ecs:${var.aws_region}:${local.account_id}:task/${aws_ecs_cluster.adapter.name}/*"
      },
      {
        Sid       = "PassRunExecutionRole"
        Effect    = "Allow"
        Action    = "iam:PassRole"
        Resource  = aws_iam_role.run_execution.arn
        Condition = { StringEquals = { "iam:PassedToService" = "ecs-tasks.amazonaws.com" } }
      },
      {
        Sid      = "RunKeys"
        Effect   = "Allow"
        Action   = "sts:AssumeRole"
        Resource = aws_iam_role.run_files.arn
      },
      {
        Sid      = "UsersAWSRoles"
        Effect   = "Allow"
        Action   = "sts:AssumeRole"
        Resource = "arn:aws:iam::*:role/AllowClutterToDeployTerraformRole-*"
      },
      {
        Sid      = "RunObjects"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.runs.arn}/runs/*"
      },
      {
        Sid      = "RunLogs"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject"]
        Resource = "${aws_s3_bucket.runs.arn}/logs/*"
      },
      # Function code users upload (internal/codestore). No lifecycle rule
      # covers code/: B deletes a zip once no function_code row names it, at
      # once on a replace or remove, and through its hourly sweep after a
      # delete in A.
      {
        Sid      = "FunctionCode"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.runs.arn}/code/*"
      },
      # Without s3:ListBucket, S3 answers a read of a missing object with 403,
      # not 404, and B could not tell a queued run's missing log from a fault.
      # No prefix condition: this role already reads every key in the bucket
      # (runs/, logs/ and code/), so listing it shows nothing new.
      {
        Sid      = "ListRunBucket"
        Effect   = "Allow"
        Action   = "s3:ListBucket"
        Resource = aws_s3_bucket.runs.arn
      },
      {
        Sid      = "AWSTemplates"
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "${var.templates_bucket_arn}/templates/aws/*"
      },
      {
        Sid       = "ListAWSTemplates"
        Effect    = "Allow"
        Action    = "s3:ListBucket"
        Resource  = var.templates_bucket_arn
        Condition = { StringLike = { "s3:prefix" = "templates/aws/*" } }
      },
    ]
  })
}

# --- The per-run files role ---

resource "aws_iam_role" "run_files" {
  name                 = "ClutterAdapterRunFilesRole"
  max_session_duration = 3600

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = aws_iam_role.runner.arn }
      Action    = "sts:AssumeRole"
    }]
  })
}

# The ceiling for every run; B's session policy narrows it to one run id and
# one deployment id (internal/runtask/ecs.go, runPolicy).
resource "aws_iam_role_policy" "run_files" {
  name = "adapter-run-files"
  role = aws_iam_role.run_files.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "RunInputs"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.runs.arn}/runs/*/in/*"
      },
      {
        Sid      = "RunOutputs"
        Effect   = "Allow"
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.runs.arn}/runs/*/out/*"
      },
      {
        Sid       = "ListRunInputs"
        Effect    = "Allow"
        Action    = "s3:ListBucket"
        Resource  = aws_s3_bucket.runs.arn
        Condition = { StringLike = { "s3:prefix" = "runs/*" } }
      },
      {
        Sid       = "ListState"
        Effect    = "Allow"
        Action    = "s3:ListBucket"
        Resource  = aws_s3_bucket.state.arn
        Condition = { StringLike = { "s3:prefix" = "state/*" } }
      },
      {
        Sid      = "State"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject"]
        Resource = "${aws_s3_bucket.state.arn}/state/*/terraform.tfstate"
      },
      {
        Sid      = "StateLock"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.state.arn}/state/*/terraform.tfstate.tflock"
      },
    ]
  })
}

# --- Execution roles (used by ECS itself, never by the containers) ---

resource "aws_iam_role" "api_execution" {
  name               = "ClutterAdapterApiExecutionRole"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
}

resource "aws_iam_role_policy_attachment" "api_execution" {
  role       = aws_iam_role.api_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy" "api_execution_secrets" {
  name = "adapter-api-secrets"
  role = aws_iam_role.api_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "secretsmanager:GetSecretValue"
      Resource = [aws_secretsmanager_secret.database_url.arn, aws_secretsmanager_secret.jwt_secret.arn]
    }]
  })
}

resource "aws_iam_role" "run_execution" {
  name               = "ClutterAdapterRunExecutionRole"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
}

resource "aws_iam_role_policy_attachment" "run_execution" {
  role       = aws_iam_role.run_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# ECS reads a run's task.env (its environmentFiles override) with this role.
resource "aws_iam_role_policy" "run_execution_env_file" {
  name = "adapter-run-env-file"
  role = aws_iam_role.run_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.runs.arn}/runs/*/in/task.env"
      },
      {
        Effect   = "Allow"
        Action   = "s3:GetBucketLocation"
        Resource = aws_s3_bucket.runs.arn
      },
    ]
  })
}
