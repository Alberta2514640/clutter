output "cluster_arn" {
  description = "ARN of the ECS cluster"
  value       = aws_ecs_cluster.main.arn
}

output "ansible_task_def_arn" {
  description = "ARN of the Ansible runner ECS task definition"
  value       = aws_ecs_task_definition.ansible_runner.arn
}

output "ansible_sg_id" {
  description = "Security group ID for the Ansible runner Fargate task"
  value       = aws_security_group.ansible_runner.id
}

output "ansible_ecr_repository_url" {
  description = "URL of the Ansible runner ECR repository"
  value       = aws_ecr_repository.ansible_runner.repository_url
}

output "ansible_task_role_arn" {
  description = "ARN of the Ansible runner Fargate task role (for iam:PassRole scoping)"
  value       = aws_iam_role.ansible_runner_task.arn
}

output "ecs_execution_role_arn" {
  description = "ARN of the ECS task execution role (for iam:PassRole scoping)"
  value       = aws_iam_role.ecs_execution.arn
}
