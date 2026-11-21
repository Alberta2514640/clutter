output "repository_url" {
  description = "ECR repository of the website's image"
  value       = aws_ecr_repository.site.repository_url
}
