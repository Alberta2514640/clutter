variable "aws_region" {
  type        = string
  description = "AWS region for bucket naming"
}

variable "enable_versioning" {
  type        = bool
  description = "Enable versioning for the S3 bucket"
  default     = true
}

variable "kms_key_arn" {
  type        = string
  description = "ARN of KMS key for encryption (leave empty to use AWS managed key)"
  default     = null
}

variable "enable_cors" {
  type        = bool
  description = "Enable CORS configuration for frontend access"
  default     = true
}

variable "cors_allowed_origins" {
  type        = list(string)
  description = "List of allowed origins for CORS"
  default     = ["*"]
}
