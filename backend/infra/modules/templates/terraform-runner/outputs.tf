# =============================================================================
# Outputs
# =============================================================================

# ECR
output "ecr_repository_url" {
  description = "ECR repository URL for the Terraform runner image"
  value       = aws_ecr_repository.terraform_runner.repository_url
}

output "ecr_repository_arn" {
  description = "ECR repository ARN"
  value       = aws_ecr_repository.terraform_runner.arn
}

# ECS
output "ecs_cluster_arn" {
  description = "ECS cluster ARN"
  value       = aws_ecs_cluster.terraform_runner.arn
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.terraform_runner.name
}

output "task_definition_arn" {
  description = "ECS task definition ARN"
  value       = aws_ecs_task_definition.terraform_runner.arn
}

output "task_definition_family" {
  description = "ECS task definition family"
  value       = aws_ecs_task_definition.terraform_runner.family
}

# VPC
output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.terraform_runner.id
}

# Note: Using public subnets for Fargate tasks (no NAT Gateway )
output "fargate_subnet_ids" {
  description = "Subnet IDs for Fargate tasks (public subnets)"
  value       = aws_subnet.public[*].id
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = aws_subnet.public[*].id
}

output "security_group_id" {
  description = "Security group ID for Fargate tasks"
  value       = aws_security_group.fargate_tasks.id
}

# IAM
output "task_execution_role_arn" {
  description = "ECS task execution role ARN"
  value       = aws_iam_role.ecs_task_execution.arn
}

output "task_role_arn" {
  description = "ECS task role ARN"
  value       = aws_iam_role.ecs_task.arn
}

output "deploy_trigger_lambda_role_arn" {
  description = "Deploy trigger Lambda execution role ARN"
  value       = aws_iam_role.deploy_trigger_lambda.arn
}

output "log_writer_lambda_role_arn" {
  description = "Log writer Lambda execution role ARN"
  value       = aws_iam_role.log_writer_lambda.arn
}

# CloudWatch
output "cloudwatch_log_group_name" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.terraform_runner.name
}

output "cloudwatch_log_group_arn" {
  description = "CloudWatch log group ARN"
  value       = aws_cloudwatch_log_group.terraform_runner.arn
}
