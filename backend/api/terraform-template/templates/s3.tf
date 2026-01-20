# Resource Documentation: https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket

# ============================================
# Variables
# ============================================
variable "bucket_name" {
  type        = string
  description = "Unique name for the S3 bucket (must be globally unique across AWS)"
}

variable "force_destroy" {
  type        = bool
  default     = false
  description = "If true, all objects are deleted when the bucket is destroyed"
}

variable "enable_versioning" {
  type        = bool
  default     = true
  description = "Enable versioning to keep multiple variants of objects"
}

variable "sse_algorithm" {
  type        = string
  default     = "AES256"
  description = "Server-side encryption algorithm (AES256 or aws:kms)"
}

variable "kms_key_arn" {
  type        = string
  default     = null
  description = "ARN of the KMS key for encryption (required if sse_algorithm is aws:kms)"
}

variable "block_public_access" {
  type        = bool
  default     = true
  description = "Block all public access to the bucket"
}

variable "object_ownership" {
  type        = string
  default     = "BucketOwnerEnforced"
  description = "Object ownership setting (BucketOwnerEnforced, BucketOwnerPreferred, ObjectWriter)"
}

variable "enable_cors" {
  type        = bool
  default     = false
  description = "Enable CORS configuration for cross-origin access"
}

variable "cors_allowed_origins" {
  type        = list(string)
  default     = ["*"]
  description = "List of origins allowed to make cross-origin requests"
}

variable "cors_allowed_methods" {
  type        = list(string)
  default     = ["GET", "PUT", "POST", "DELETE", "HEAD"]
  description = "HTTP methods allowed for CORS requests"
}

variable "cors_allowed_headers" {
  type        = list(string)
  default     = ["*"]
  description = "Headers allowed in CORS preflight requests"
}

variable "cors_expose_headers" {
  type        = list(string)
  default     = ["ETag"]
  description = "Headers exposed to the browser in CORS responses"
}

variable "cors_max_age_seconds" {
  type        = number
  default     = 3000
  description = "Time in seconds that browser can cache the preflight response"
}

variable "enable_logging" {
  type        = bool
  default     = false
  description = "Enable server access logging"
}

variable "logging_target_bucket" {
  type        = string
  default     = null
  description = "Target bucket for access logs (required if enable_logging is true)"
}

variable "logging_target_prefix" {
  type        = string
  default     = "logs/"
  description = "Prefix for log object keys"
}

variable "enable_lifecycle" {
  type        = bool
  default     = false
  description = "Enable lifecycle rules for automatic object management"
}

variable "lifecycle_expiration_days" {
  type        = number
  default     = 90
  description = "Number of days until objects expire"
}

variable "lifecycle_noncurrent_expiration_days" {
  type        = number
  default     = 30
  description = "Number of days until noncurrent versions expire"
}

variable "lifecycle_abort_incomplete_days" {
  type        = number
  default     = 7
  description = "Number of days to abort incomplete multipart uploads"
}

variable "tags" {
  type        = map(string)
  default     = {}
  description = "Tags to apply to the S3 bucket"
}

# ============================================
# Resources
# ============================================
resource "aws_s3_bucket" "this" {
  bucket        = var.bucket_name
  force_destroy = var.force_destroy
  tags          = var.tags
}

resource "aws_s3_bucket_public_access_block" "this" {
  bucket                  = aws_s3_bucket.this.id
  block_public_acls       = var.block_public_access
  block_public_policy     = var.block_public_access
  ignore_public_acls      = var.block_public_access
  restrict_public_buckets = var.block_public_access
}

resource "aws_s3_bucket_versioning" "this" {
  bucket = aws_s3_bucket.this.id
  versioning_configuration {
    status = var.enable_versioning ? "Enabled" : "Suspended"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "this" {
  bucket = aws_s3_bucket.this.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = var.kms_key_arn != null ? "aws:kms" : var.sse_algorithm
      kms_master_key_id = var.kms_key_arn
    }
    bucket_key_enabled = var.kms_key_arn != null
  }
}

resource "aws_s3_bucket_ownership_controls" "this" {
  bucket = aws_s3_bucket.this.id
  rule {
    object_ownership = var.object_ownership
  }
}

# Conditional CORS Configuration
resource "aws_s3_bucket_cors_configuration" "this" {
  count  = var.enable_cors ? 1 : 0
  bucket = aws_s3_bucket.this.id
  cors_rule {
    allowed_headers = var.cors_allowed_headers
    allowed_methods = var.cors_allowed_methods
    allowed_origins = var.cors_allowed_origins
    expose_headers  = var.cors_expose_headers
    max_age_seconds = var.cors_max_age_seconds
  }
}

# Conditional Logging Configuration
resource "aws_s3_bucket_logging" "this" {
  count         = var.enable_logging ? 1 : 0
  bucket        = aws_s3_bucket.this.id
  target_bucket = var.logging_target_bucket
  target_prefix = var.logging_target_prefix
}

# Conditional Lifecycle Configuration
resource "aws_s3_bucket_lifecycle_configuration" "this" {
  count      = var.enable_lifecycle ? 1 : 0
  bucket     = aws_s3_bucket.this.id
  depends_on = [aws_s3_bucket_versioning.this]
  rule {
    id     = "cleanup-rule"
    status = "Enabled"
    expiration {
      days = var.lifecycle_expiration_days
    }
    noncurrent_version_expiration {
      noncurrent_days = var.lifecycle_noncurrent_expiration_days
    }
    abort_incomplete_multipart_upload {
      days_after_initiation = var.lifecycle_abort_incomplete_days
    }
  }
}

# ============================================
# Outputs
# ============================================
output "bucket_id" {
  value       = aws_s3_bucket.this.id
  description = "ID of the S3 bucket"
}

output "bucket_arn" {
  value       = aws_s3_bucket.this.arn
  description = "ARN of the S3 bucket"
}

output "bucket_domain_name" {
  value       = aws_s3_bucket.this.bucket_domain_name
  description = "Domain name of the S3 bucket"
}

output "bucket_regional_domain_name" {
  value       = aws_s3_bucket.this.bucket_regional_domain_name
  description = "Regional domain name of the S3 bucket"
}
