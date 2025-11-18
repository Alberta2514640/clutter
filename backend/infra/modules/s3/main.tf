resource "random_id" "suffix" {
  byte_length = 4
}

resource "aws_s3_bucket" "iac_storage" {
  bucket = "clutter-${var.aws_region}-${random_id.suffix.hex}"

  tags = merge(
    var.tags,
    {
      Name        = "clutter-${var.aws_region}-${random_id.suffix.hex}"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  )
}

# Block all public access
resource "aws_s3_bucket_public_access_block" "iac_storage" {
  bucket = aws_s3_bucket.iac_storage.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Versioning for data protection
resource "aws_s3_bucket_versioning" "iac_storage" {
  bucket = aws_s3_bucket.iac_storage.id

  versioning_configuration {
    status = var.enable_versioning ? "Enabled" : "Suspended"
  }
}

# Server-side encryption (AES256 or KMS)
resource "aws_s3_bucket_server_side_encryption_configuration" "iac_storage" {
  bucket = aws_s3_bucket.iac_storage.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = var.kms_key_arn != null ? "aws:kms" : "AES256"
      kms_master_key_id = var.kms_key_arn
    }
    bucket_key_enabled = var.kms_key_arn != null ? true : false
  }
}

# CORS for frontend access
resource "aws_s3_bucket_cors_configuration" "iac_storage" {
  count  = var.enable_cors ? 1 : 0
  bucket = aws_s3_bucket.iac_storage.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "PUT", "POST", "DELETE", "HEAD"]
    allowed_origins = var.cors_allowed_origins
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}

# Bucket policy - enforce HTTPS and encryption
resource "aws_s3_bucket_policy" "iac_storage" {
  bucket = aws_s3_bucket.iac_storage.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.iac_storage.arn,
          "${aws_s3_bucket.iac_storage.arn}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      },
      {
        Sid       = "DenyUnencryptedObjectUploads"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:PutObject"
        Resource  = "${aws_s3_bucket.iac_storage.arn}/*"
        Condition = {
          StringNotEquals = {
            "s3:x-amz-server-side-encryption" = var.kms_key_arn != null ? "aws:kms" : "AES256"
          }
        }
      }
    ]
  })
}

# Object ownership
resource "aws_s3_bucket_ownership_controls" "iac_storage" {
  bucket = aws_s3_bucket.iac_storage.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}
