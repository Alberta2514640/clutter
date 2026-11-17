output "load_balancer_dns_name" {
  description = "Point the site's one name (the certificate's) at this with a CNAME or alias record"
  value       = aws_lb.adapter.dns_name
}

output "vpc_id" {
  description = "B's VPC, which the website shares for the load balancer"
  value       = aws_vpc.adapter.id
}

output "public_subnet_ids" {
  description = "B's public subnets"
  value       = aws_subnet.public[*].id
}

output "alb_security_group_id" {
  description = "The load balancer's security group"
  value       = aws_security_group.alb.id
}

output "https_listener_arn" {
  description = "The load balancer's HTTPS listener, which routes by path"
  value       = aws_lb_listener.https.arn
}

output "api_repository_url" {
  description = "ECR repository of B API's image"
  value       = aws_ecr_repository.adapter["clutter-adapter-api"].repository_url
}

output "runtask_repository_url" {
  description = "ECR repository of the run task's image"
  value       = aws_ecr_repository.adapter["clutter-adapter-runtask"].repository_url
}

output "runner_role_arn" {
  description = "B API's task role, which users' AWS roles trust (AdapterRunnerRoleName)"
  value       = aws_iam_role.runner.arn
}
