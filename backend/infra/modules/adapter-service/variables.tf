variable "aws_region" {
  description = "The platform region (ca-central-1, ADR-015)"
  type        = string
}

variable "image_tag" {
  description = "Version tag of B's two images (clutter-adapter-api and clutter-adapter-runtask), pushed by backend/adapter/images/build-and-push.sh"
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$", var.image_tag)) && var.image_tag != "latest"
    error_message = "image_tag must be a version tag such as v1, never latest."
  }
}

variable "certificate_arn" {
  description = "ARN of the ACM certificate, in the platform region, for the site's one name (domain)"
  type        = string

  validation {
    condition     = can(regex("^arn:aws[a-z-]*:acm:[a-z0-9-]+:[0-9]{12}:certificate/[A-Za-z0-9-]+$", var.certificate_arn))
    error_message = "certificate_arn must be an ACM certificate ARN."
  }
}

variable "domain" {
  description = "The site's one DNS name, such as app.example.com; the load balancer sends its /v1/* and /healthz to B API"
  type        = string

  validation {
    condition     = can(regex("^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$", var.domain)) && length(var.domain) <= 128
    error_message = "domain must be a lower-case DNS name such as app.example.com, with no scheme, port or path."
  }
}

variable "frontend_url" {
  description = "The site's origin; B answers CORS for it alone (ADAPTER_ALLOWED_ORIGINS)"
  type        = string
}

variable "database_url" {
  description = "Postgres connection string, the same database as A"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "The secret A's log-in signs tokens with"
  type        = string
  sensitive   = true
}

variable "templates_bucket_name" {
  description = "A's template bucket; the AWS connector reads templates/aws/ from it"
  type        = string
}

variable "templates_bucket_arn" {
  description = "ARN of A's template bucket"
  type        = string
}

variable "runner_role_name" {
  description = "Name of B's API task role, which assumes users' AWS roles. It must match AdapterRunnerRoleName in modules/s3/uploads/client_side_terraform_deployer_role.yaml"
  type        = string
  default     = "ClutterAdapterRunnerRole"
}

variable "secrets_prefix" {
  description = "Start of every connection secret's name in Secrets Manager (ADAPTER_SECRETS_PREFIX)"
  type        = string
  default     = "clutter/adapter/connections/"
}

variable "vpc_cidr" {
  description = "Address range of B's own VPC"
  type        = string
  default     = "10.42.0.0/16"
}

variable "run_timeout" {
  description = "Longest a run may last (ADAPTER_RUN_TIMEOUT). Under one hour, because a run's keys last at most one hour"
  type        = string
  default     = "55m"
}

variable "run_task_cpu" {
  description = "CPU units of each run task"
  type        = number
  default     = 1024
}

variable "run_task_memory" {
  description = "Memory (MiB) of each run task"
  type        = number
  default     = 2048
}
