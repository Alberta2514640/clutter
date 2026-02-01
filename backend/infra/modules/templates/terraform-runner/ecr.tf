# =============================================================================
# ECR Repository for Terraform Runner Image
# =============================================================================

resource "aws_ecr_repository" "terraform_runner" {
  name                 = "clutter-terraform-runner"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = {
    Name        = "clutter-terraform-runner"
    Environment = var.environment
  }
}

# Lifecycle policy to clean up old images
resource "aws_ecr_lifecycle_policy" "terraform_runner" {
  repository = aws_ecr_repository.terraform_runner.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep last 10 images"
        selection = {
          tagStatus     = "any"
          countType     = "imageCountMoreThan"
          countNumber   = 10
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}
