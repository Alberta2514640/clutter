# The website's own cluster (never B's "clutter-adapter": B API's role may stop any
# task in B's cluster) and its service behind the shared load balancer.

resource "aws_ecs_cluster" "site" {
  name = "clutter-website"
}

resource "aws_cloudwatch_log_group" "site" {
  name              = "/ecs/clutter-website"
  retention_in_days = 30
}

resource "aws_ecs_task_definition" "site" {
  family                   = "clutter-website"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.cpu
  memory                   = var.memory
  execution_role_arn       = aws_iam_role.execution.arn
  # No task_role_arn: the website calls no AWS API.

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }

  container_definitions = jsonencode([{
    name         = "site"
    image        = "${aws_ecr_repository.site.repository_url}:${var.image_tag}"
    essential    = true
    portMappings = [{ containerPort = 3000, protocol = "tcp" }]
    # The image sets PORT and HOSTNAME too; these keep the server on every
    # interface whatever Fargate names the host.
    environment = [
      { name = "HOSTNAME", value = "0.0.0.0" },
      { name = "PORT", value = "3000" },
    ]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.site.name
        awslogs-region        = var.aws_region
        awslogs-stream-prefix = "site"
      }
    }
  }])
}

# A deploy starts the new task before it stops the old one, so the site stays
# up. Until the image exists in ECR (the first deploy), ECS keeps retrying the
# task; npm run deploy -- prod --site-image pushes it.
resource "aws_ecs_service" "site" {
  name                               = "clutter-website"
  cluster                            = aws_ecs_cluster.site.id
  task_definition                    = aws_ecs_task_definition.site.arn
  desired_count                      = 1
  launch_type                        = "FARGATE"
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 60

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [aws_security_group.site.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.site.arn
    container_name   = "site"
    container_port   = 3000
  }

  # The target group must be attached to the listener before a service uses it.
  depends_on = [aws_lb_listener_rule.site]
}
