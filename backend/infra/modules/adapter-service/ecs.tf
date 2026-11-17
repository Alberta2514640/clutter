# B's own cluster (never A's "clutter"), B API's service behind the load
# balancer, and the run task B starts once per run.

resource "aws_ecs_cluster" "adapter" {
  name = "clutter-adapter"
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/ecs/clutter-adapter-api"
  retention_in_days = 30
}

# Only the run task's own status lines: Terraform's output goes to the run
# log object, which B redacts before storing.
resource "aws_cloudwatch_log_group" "run" {
  name              = "/ecs/clutter-adapter-run"
  retention_in_days = 14
}

locals {
  api_environment = {
    AWS_REGION                   = var.aws_region
    ADAPTER_CREDENTIAL_STORE     = "secretsmanager"
    ADAPTER_SECRETS_PREFIX       = var.secrets_prefix
    ADAPTER_CONNECTORS           = "general-openstack,aws"
    ADAPTER_AWS_TEMPLATE_BUCKET  = var.templates_bucket_name
    ADAPTER_AWS_STS              = "sts"
    ADAPTER_ALLOWED_ORIGINS      = trimsuffix(var.frontend_url, "/")
    ADAPTER_RUN_TIMEOUT          = var.run_timeout
    ADAPTER_RUN_LAUNCHER         = "ecs"
    ADAPTER_RUN_CLUSTER          = aws_ecs_cluster.adapter.name
    ADAPTER_RUN_TASK_DEFINITION  = aws_ecs_task_definition.run.arn
    ADAPTER_RUN_CONTAINER        = "run"
    ADAPTER_RUN_SUBNETS          = join(",", aws_subnet.run[*].id)
    ADAPTER_RUN_SECURITY_GROUPS  = aws_security_group.run.id
    ADAPTER_RUN_ASSIGN_PUBLIC_IP = "true"
    ADAPTER_RUN_BUCKET           = aws_s3_bucket.runs.bucket
    ADAPTER_STATE_BUCKET         = aws_s3_bucket.state.bucket
    ADAPTER_RUN_FILES_ROLE_ARN   = aws_iam_role.run_files.arn
    # Run logs outlive B API's task: objects under logs/ in the run bucket,
    # outside the runs/ prefix that expires after a day.
    ADAPTER_LOG_STORE  = "s3"
    ADAPTER_LOG_BUCKET = aws_s3_bucket.runs.bucket
  }
}

resource "aws_ecs_task_definition" "api" {
  family                   = "clutter-adapter-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.api_execution.arn
  task_role_arn            = aws_iam_role.runner.arn

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }

  container_definitions = jsonencode([{
    name         = "api"
    image        = "${aws_ecr_repository.adapter["clutter-adapter-api"].repository_url}:${var.image_tag}"
    essential    = true
    portMappings = [{ containerPort = 8080, protocol = "tcp" }]
    environment  = [for k in sort(keys(local.api_environment)) : { name = k, value = local.api_environment[k] }]
    secrets = [
      { name = "ADAPTER_DATABASE_URL", valueFrom = aws_secretsmanager_secret.database_url.arn },
      { name = "JWT_SECRET", valueFrom = aws_secretsmanager_secret.jwt_secret.arn },
    ]
    # B stops within 30 seconds of SIGTERM (backend/adapter/README.md).
    stopTimeout = 35
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.api.name
        awslogs-region        = var.aws_region
        awslogs-stream-prefix = "api"
      }
    }
  }])

  depends_on = [aws_secretsmanager_secret_version.database_url, aws_secretsmanager_secret_version.jwt_secret]
}

resource "aws_ecs_task_definition" "run" {
  family                   = local.run_family
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.run_task_cpu
  memory                   = var.run_task_memory
  execution_role_arn       = aws_iam_role.run_execution.arn
  # No task_role_arn: a run task holds only the keys B hands it for that run.

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }

  container_definitions = jsonencode([{
    name      = "run"
    image     = "${aws_ecr_repository.adapter["clutter-adapter-runtask"].repository_url}:${var.image_tag}"
    essential = true
    # Fargate's longest; the run task gives Terraform 100 seconds to release
    # the state lock after it is stopped.
    stopTimeout = 120
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.run.name
        awslogs-region        = var.aws_region
        awslogs-stream-prefix = "run"
      }
    }
  }])
}

# One copy only: B's job queue is in-process, and at start-up B fails any run
# a previous copy left running. A deploy therefore stops the old task before
# the new one starts (a short outage), and never runs two at once.
resource "aws_ecs_service" "api" {
  name                               = "clutter-adapter-api"
  cluster                            = aws_ecs_cluster.adapter.id
  task_definition                    = aws_ecs_task_definition.api.arn
  desired_count                      = 1
  launch_type                        = "FARGATE"
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100
  health_check_grace_period_seconds  = 60

  network_configuration {
    subnets          = aws_subnet.public[*].id
    security_groups  = [aws_security_group.api.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.api.arn
    container_name   = "api"
    container_port   = 8080
  }

  # The target group must be attached to the listener before a service uses it.
  depends_on = [aws_lb_listener_rule.api]
}
