resource "aws_ecr_repository" "terraform_deployer" {
  name = "clutter-terraform-deployer"
}

resource "aws_iam_role" "ecs_execution" {
  name = "ecsTaskExecutionRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action = "sts:AssumeRole"
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
        Effect = "Allow"
        Action = "sts:AssumeRole"
        Resource = "arn:aws:iam::*:role/AllowClutterToDeployTerraformRole-*"
      },
      {
        Effect = "Allow"
        Action = "s3:ListBucket"
        Resource = var.s3_clutter
      },
      {
        Effect = "Allow"
        Action = "s3:GetObject"
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

