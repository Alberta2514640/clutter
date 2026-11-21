variable "aws_region" {
  description = "The platform region (ca-central-1, ADR-015)"
  type        = string
}

variable "image_tag" {
  description = "Version tag of the website's image (clutter-website), pushed by npm run deploy -- prod --site-image"
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$", var.image_tag)) && var.image_tag != "latest"
    error_message = "image_tag must be a version tag such as v1, never latest."
  }
}

variable "domain" {
  description = "The site's one DNS name, such as app.example.com; the load balancer sends it to the website, except B's paths"
  type        = string

  validation {
    condition     = can(regex("^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$", var.domain)) && length(var.domain) <= 128
    error_message = "domain must be a lower-case DNS name such as app.example.com, with no scheme, port or path."
  }
}

variable "vpc_id" {
  description = "B's VPC, which holds the shared load balancer"
  type        = string
}

variable "subnet_ids" {
  description = "B's public subnets; the website's tasks run there with public addresses (no NAT)"
  type        = list(string)
}

variable "alb_security_group_id" {
  description = "The shared load balancer's security group; only it may reach the website"
  type        = string
}

variable "listener_arn" {
  description = "The shared load balancer's HTTPS listener"
  type        = string
}

variable "listener_rule_priority" {
  description = "Priority of the website's rule on the HTTPS listener; it must come after B API's path rule (10)"
  type        = number
  default     = 20
}

variable "cpu" {
  description = "CPU units of each website task"
  type        = number
  default     = 256
}

variable "memory" {
  description = "Memory (MiB) of each website task"
  type        = number
  default     = 512
}
