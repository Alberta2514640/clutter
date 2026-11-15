# The website's one role: ECS uses it to pull the website's image and write
# its logs, and nothing else. The task has no task role, because the website
# calls no AWS API: the browser calls A's and B's APIs directly.

data "aws_caller_identity" "current" {}

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
      values   = [data.aws_caller_identity.current.account_id]
    }
  }
}

resource "aws_iam_role" "execution" {
  name               = "ClutterWebsiteExecutionRole"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
}

# The managed AmazonECSTaskExecutionRolePolicy would allow every repository
# and every log group; this allows only the website's own.
resource "aws_iam_role_policy" "execution" {
  name = "website-pull-and-log"
  role = aws_iam_role.execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # ECR's log-in has no resource-level permission.
        Sid      = "RegistryLogIn"
        Effect   = "Allow"
        Action   = "ecr:GetAuthorizationToken"
        Resource = "*"
      },
      {
        Sid      = "PullWebsiteImage"
        Effect   = "Allow"
        Action   = ["ecr:BatchCheckLayerAvailability", "ecr:GetDownloadUrlForLayer", "ecr:BatchGetImage"]
        Resource = aws_ecr_repository.site.arn
      },
      {
        Sid      = "WriteWebsiteLogs"
        Effect   = "Allow"
        Action   = ["logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "${aws_cloudwatch_log_group.site.arn}:*"
      },
    ]
  })
}
