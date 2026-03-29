data "aws_caller_identity" "current" {}

resource "aws_ecr_repository" "terraform_deployer" {
  name                  = "clutter-terraform-deployer"
  image_tag_mutability  = "IMMUTABLE"
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
      # ----------- CLUTTER S3 (READ + WRITE STATE) -----------
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = var.s3_clutter_arn
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${var.s3_clutter_arn}/*"
      },
      # ----------- TEMPLATES S3 (READ ONLY) -----------
      {
        Effect = "Allow"
        Action = "s3:ListBucket"
        Resource = var.s3_templates_arn
      },
      {
        Effect = "Allow"
        Action = "s3:GetObject"
        Resource = "${var.s3_templates_arn}/zip/*"
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
        { name = "TF_CLI_ARGS", value = "-no-color -input=false" },
        { name = "S3_CLUTTER_NAME", value = var.s3_clutter_name },
        { name = "S3_TEMPLATES_NAME", value = var.s3_templates_name }
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
  name                  = "clutter-ansible-runner"
  image_tag_mutability  = "IMMUTABLE"
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
      # STS — assume the client-provided IAM role in their account
      {
        Effect   = "Allow"
        Action   = "sts:AssumeRole"
        Resource = "arn:aws:iam::*:role/AllowClutterToAnsibleRole-*"
      },
      # S3 — playbook download, log upload, and SSM connection plugin file transfer
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket",
          "s3:GetBucketLocation"
        ]
        Resource = [
          var.s3_clutter_arn,
          "${var.s3_clutter_arn}/*"
        ]
      },

      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
          "ec2:StopInstances"
        ]
        Resource = "*"
      },
      # SSM Session Manager — start sessions on the SSM document (no tag condition)
      {
        Effect   = "Allow"
        Action   = "ssm:StartSession"
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:document/SSM-SessionManagerRunShell"
      },
      # SSM Session Manager — start sessions on EC2 instances tagged ManagedBy=Ansible
      {
        Effect   = "Allow"
        Action   = "ssm:StartSession"
        Resource = "arn:aws:ec2:${var.aws_region}:${data.aws_caller_identity.current.account_id}:instance/*"
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
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:session/*"
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:DescribeSessions",
          "ssm:DescribeInstanceInformation",
          "ssm:GetConnectionStatus"
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
        { name = "S3_BUCKET_NAME", value = var.s3_clutter_name },
        { name = "AWS_DEFAULT_REGION", value = var.aws_region },
        { name = "PSQL_CONNECTION_STRING", value = var.psql_connection_string }
      ]

      essential = true
    }
  ])
}

resource "aws_security_group" "ansible_runner" {
  name   = "ansible-runner"
  vpc_id = data.aws_vpc.default.id

  # DNS egress for AWS and database hostname resolution.
  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

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
