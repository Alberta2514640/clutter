resource "random_id" "suffix" {
  byte_length = 4
}

# ================================================================================================================================
# Bucket - Clutter
# Private
# Description: Will house user diagram generated Terraform code and Ansible playbooks
# ================================================================================================================================
resource "aws_s3_bucket" "clutter_bucket" {
  bucket = "clutter-${var.aws_region}-${random_id.suffix.hex}"
}

# Block all public access
resource "aws_s3_bucket_public_access_block" "clutter_bucket" {
  bucket = aws_s3_bucket.clutter_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Versioning for data protection
resource "aws_s3_bucket_versioning" "clutter_bucket" {
  bucket = aws_s3_bucket.clutter_bucket.id

  versioning_configuration {
    status = var.enable_versioning ? "Enabled" : "Suspended"
  }
}

# Server-side encryption (AES256 or KMS)
resource "aws_s3_bucket_server_side_encryption_configuration" "clutter_bucket" {
  bucket = aws_s3_bucket.clutter_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = var.kms_key_arn != null ? "aws:kms" : "AES256"
      kms_master_key_id = var.kms_key_arn
    }
    bucket_key_enabled = var.kms_key_arn != null ? true : false
  }
}

# CORS for frontend access
resource "aws_s3_bucket_cors_configuration" "clutter_bucket" {
  count  = var.enable_cors ? 1 : 0
  bucket = aws_s3_bucket.clutter_bucket.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "PUT", "POST", "DELETE", "HEAD"]
    allowed_origins = var.cors_allowed_origins
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}

# Bucket policy - enforce HTTPS and encryption
resource "aws_s3_bucket_policy" "clutter_bucket" {
  bucket = aws_s3_bucket.clutter_bucket.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.clutter_bucket.arn,
          "${aws_s3_bucket.clutter_bucket.arn}/*"
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
        Resource  = "${aws_s3_bucket.clutter_bucket.arn}/*"
        Condition = {
          StringNotEquals = {
            "s3:x-amz-server-side-encryption": "AES256"
          }
        }
      }
    ]
  })
}

# Object ownership
resource "aws_s3_bucket_ownership_controls" "clutter_bucket" {
  bucket = aws_s3_bucket.clutter_bucket.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

# ================================================================================================================================
# Bucket - Templates
# Public
# Description: Will house generic Terraformr esource templates for use by the generator and other public templates
# ================================================================================================================================
resource "aws_s3_bucket" "templates" {
  bucket = "clutter-templates-${var.aws_region}-${random_id.suffix.hex}"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "templates_encryption" {
  bucket = aws_s3_bucket.templates.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "templates_public_access" {
  bucket = aws_s3_bucket.templates.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "templates_policy" {
  bucket = aws_s3_bucket.templates.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = "*"
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.templates.arn}/*"
    }]
  })
}


# Object - Client-side Terraform Deployer Role Template
resource "aws_s3_object" "terraform_deployer_role_template" {

  bucket = aws_s3_bucket.templates.id

  key = "templates/cloudformation/client_side_terraform_deployer_role.yaml"
  source = "${path.module}/uploads/client_side_terraform_deployer_role.yaml"

  server_side_encryption = "AES256"

  etag = filemd5("${path.module}/uploads/client_side_terraform_deployer_role.yaml")

}
