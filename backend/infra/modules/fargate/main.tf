data "aws_caller_identity" "current" {}

resource "aws_ecr_repository" "terraform_deployer" {
  name = "clutter-terraform-deployer"
}

resource "aws_iam_role" "ecs_execution" {
  name = "ecsTaskExecutionRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_execution_attach" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}


resource "aws_iam_role" "terraform_deployer_task" {
  name = "TerraformDeployerFargateTaskRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })
}

resource "aws_iam_policy" "terraform_deployer_task_policy" {
  name = "TerraformDeployerFargateTaskPolicy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "sts:AssumeRole"
        Resource = "arn:aws:iam::*:role/AllowClutterToDeployTerraformRole-*"
      },
      {
        Effect   = "Allow"
        Action   = "s3:ListBucket"
        Resource = var.s3_clutter
      },
      {
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "${var.s3_clutter}/*"
      }
    ]
  })
}


resource "aws_ecs_cluster" "main" {
  name = "clutter"
}

resource "aws_iam_role_policy_attachment" "terraform_task_attach" {
  role       = aws_iam_role.terraform_deployer_task.name
  policy_arn = aws_iam_policy.terraform_deployer_task_policy.arn
}

resource "aws_cloudwatch_log_group" "terraform_deployer" {
  name              = "/ecs/terraform-deployer"
  retention_in_days = 7
}

resource "aws_ecs_task_definition" "terraform_deployer" {
  family                   = "terraform-deployer"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512

  execution_role_arn = aws_iam_role.ecs_execution.arn
  task_role_arn      = aws_iam_role.terraform_deployer_task.arn

  container_definitions = jsonencode([
    {
      name  = "terraform-deployer"
      image = "${aws_ecr_repository.terraform_deployer.repository_url}:latest"

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "/ecs/terraform-deployer"
          awslogs-region        = "us-west-2"
          awslogs-stream-prefix = "ecs"
        }
      }

      environment = [
        { name = "TF_IN_AUTOMATION", value = "true" },
        { name = "TF_CLI_ARGS", value = "-no-color -input=false" }
      ]

      essential = true
    }
  ])
}

data "aws_vpc" "default" {
  default = true
}

resource "aws_security_group" "terraform_deployer" {
  name   = "terraform-deployer"
  vpc_id = data.aws_vpc.default.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ===============================================================
# Ansible Runner — ECR, IAM, Task Definition, Logs, Security Group
# ===============================================================

resource "aws_ecr_repository" "ansible_runner" {
  name = "clutter-ansible-runner"
}

resource "aws_iam_role" "ansible_runner_task" {
  name = "AnsibleRunnerFargateTaskRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_policy" "ansible_runner_task_policy" {
  name = "AnsibleRunnerFargateTaskPolicy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:ListBucket"
        ]
        Resource = [
          var.s3_clutter,
          "${var.s3_clutter}/*"
        ]
      },

      {
        Effect   = "Allow"
        Action   = "ec2:DescribeInstances"
        Resource = "*"
      },
      # SSM Session Manager — start sessions on the SSM document
      {
        Effect = "Allow"
        Action = [
          "ssm:StartSession"
        ]
        Resource = [
          "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:document/SSM-SessionManagerRunShell"
        ]
      },
      # SSM Session Manager — start sessions on tagged instances
      {
        Effect = "Allow"
        Action = [
          "ssm:StartSession"
        ]
        Resource = [
          "arn:aws:ec2:${var.aws_region}:${data.aws_caller_identity.current.account_id}:instance/*"
        ]
        Condition = {
          StringEquals = {
            "ssm:resourceTag/ManagedBy" = "Ansible"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:TerminateSession",
          "ssm:ResumeSession"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:session/$${aws:userid}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:DescribeSessions",
          "ssm:DescribeInstanceInformation"
        ]
        Resource = "*"
      },
      # SSM Session Manager data channel — required for the WebSocket
      # transport that community.aws.aws_ssm connection plugin uses
      {
        Effect = "Allow"
        Action = [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ansible_runner_task_attach" {
  role       = aws_iam_role.ansible_runner_task.name
  policy_arn = aws_iam_policy.ansible_runner_task_policy.arn
}

resource "aws_cloudwatch_log_group" "ansible_runner" {
  name              = "/ecs/ansible-runner"
  retention_in_days = 14
}

# ===============================================================
# Secrets Manager — PSQL connection string for Ansible Fargate tasks
# ===============================================================

resource "aws_secretsmanager_secret" "psql_connection_string" {
  name                    = "clutter/ansible-runner/psql-connection-string"
  description             = "PostgreSQL connection string for Ansible Runner Fargate tasks"
  recovery_window_in_days = 7
}

resource "aws_secretsmanager_secret_version" "psql_connection_string" {
  secret_id     = aws_secretsmanager_secret.psql_connection_string.id
  secret_string = var.psql_connection_string
}

# Grant ECS execution role permission to read the secret
resource "aws_iam_role_policy" "ecs_execution_secrets" {
  name = "ecs-execution-secrets-access"
  role = aws_iam_role.ecs_execution.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "secretsmanager:GetSecretValue"
      Resource = aws_secretsmanager_secret.psql_connection_string.arn
    }]
  })
}

resource "aws_ecs_task_definition" "ansible_runner" {
  family                   = "ansible-runner"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 1024
  memory                   = 2048

  execution_role_arn = aws_iam_role.ecs_execution.arn
  task_role_arn      = aws_iam_role.ansible_runner_task.arn

  container_definitions = jsonencode([
    {
      name  = "ansible-executor"
      image = "${aws_ecr_repository.ansible_runner.repository_url}:${var.ansible_runner_image_tag}"

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.ansible_runner.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }

      environment = [
        { name = "S3_BUCKET_NAME", value = var.s3_clutter_bucket_name },
        { name = "AWS_DEFAULT_REGION", value = var.aws_region }
      ]

      # Inject PSQL connection string from Secrets Manager (resolved at container start)
      secrets = [
        {
          name      = "PSQL_CONNECTION_STRING"
          valueFrom = aws_secretsmanager_secret.psql_connection_string.arn
        }
      ]

      essential = true
    }
  ])
}

resource "aws_security_group" "ansible_runner" {
  name   = "ansible-runner"
  vpc_id = data.aws_vpc.default.id

  # HTTPS egress for AWS API calls (S3, SSM Session Manager, CloudWatch)
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # PostgreSQL egress for job status updates (Supabase)
  egress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
