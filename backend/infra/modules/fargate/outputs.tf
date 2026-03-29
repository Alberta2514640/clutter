output "ecs_cluster_arn" {
  value       = aws_ecs_cluster.main.arn
}

output "ecs_cluster_name" {
  value = aws_ecs_cluster.main.name
}

output "terraform_task_definition_arn" {
  value = aws_ecs_task_definition.terraform_deployer.arn
}

output "terraform_security_group_id" {
  value = aws_security_group.terraform_deployer.id
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

output "subnet_ids" {
  value = data.aws_subnets.default.ids
}

output "ecs_execution_role_arn" {
  value = aws_iam_role.ecs_execution.arn
}

output "terraform_task_role_arn" {
  value = aws_iam_role.terraform_deployer_task.arn
}

output "terraform_container_name" {
  value = jsondecode(aws_ecs_task_definition.terraform_deployer.container_definitions)[0].name
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
  value       = aws_iam_role.terraform_deployer_task.arn
}

output "ansible_task_role_unique_id" {
  description = "Unique ID of the Ansible runner task role (for aws:userId conditions in bucket policies)"
  value       = aws_iam_role.terraform_deployer_task.unique_id
}
