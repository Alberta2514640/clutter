# B's images, its run and state buckets, and its own platform secrets.

data "aws_caller_identity" "current" {}

resource "random_id" "suffix" {
  byte_length = 4
}

# --- ECR: immutable, one version tag per build ---

resource "aws_ecr_repository" "adapter" {
  for_each             = toset(["clutter-adapter-api", "clutter-adapter-runtask"])
  name                 = each.value
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "adapter" {
  for_each   = aws_ecr_repository.adapter
  repository = each.value.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep the 20 newest images"
      selection    = { tagStatus = "any", countType = "imageCountMoreThan", countNumber = 20 }
      action       = { type = "expire" }
    }]
  })
}

# --- Run bucket: each run's files, spec, keys file, log and result, and the
# run logs users read ---
# B deletes a run's objects when the run ends; the first lifecycle rule removes
# what a crash leaves behind. The second expires run logs after 90 days.

resource "aws_s3_bucket" "runs" {
  bucket = "clutter-adapter-runs-${var.aws_region}-${random_id.suffix.hex}"
}

resource "aws_s3_bucket_lifecycle_configuration" "runs" {
  bucket = aws_s3_bucket.runs.id

  rule {
    id     = "expire-runs"
    status = "Enabled"

    filter {
      prefix = "runs/"
    }

    expiration {
      days = 1
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }

  # Run logs (ADAPTER_LOG_STORE=s3) are kept 90 days, then removed. S3 allows one
  # lifecycle configuration per bucket, so this rule lives here.
  rule {
    id     = "expire-run-logs"
    status = "Enabled"

    filter {
      prefix = "logs/"
    }

    expiration {
      days = 90
    }
  }
}

# --- State bucket: one state file and lock file per deployment ---

resource "aws_s3_bucket" "state" {
  bucket = "clutter-adapter-state-${var.aws_region}-${random_id.suffix.hex}"
}

resource "aws_s3_bucket_versioning" "state" {
  bucket = aws_s3_bucket.state.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "state" {
  bucket = aws_s3_bucket.state.id

  rule {
    id     = "expire-old-state-versions"
    status = "Enabled"

    filter {
      prefix = "state/"
    }

    noncurrent_version_expiration {
      noncurrent_days = 90
    }
  }
}

locals {
  adapter_buckets = {
    runs  = aws_s3_bucket.runs
    state = aws_s3_bucket.state
  }
}

resource "aws_s3_bucket_public_access_block" "adapter" {
  for_each                = local.adapter_buckets
  bucket                  = each.value.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "adapter" {
  for_each = local.adapter_buckets
  bucket   = each.value.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "adapter" {
  for_each = local.adapter_buckets
  bucket   = each.value.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_policy" "adapter" {
  for_each = local.adapter_buckets
  bucket   = each.value.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "DenyInsecureTransport"
      Effect    = "Deny"
      Principal = "*"
      Action    = "s3:*"
      Resource  = [each.value.arn, "${each.value.arn}/*"]
      Condition = { Bool = { "aws:SecureTransport" = "false" } }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.adapter]
}

# --- B's own secrets, read by ECS into B API's environment ---
# Users' connection secrets are created by B itself under var.secrets_prefix.

resource "aws_secretsmanager_secret" "database_url" {
  name        = "clutter/adapter/platform/database-url"
  description = "B API's Postgres connection string"
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id     = aws_secretsmanager_secret.database_url.id
  secret_string = var.database_url
}

resource "aws_secretsmanager_secret" "jwt_secret" {
  name        = "clutter/adapter/platform/jwt-secret"
  description = "The secret A's log-in signs tokens with; B verifies them"
}

resource "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id     = aws_secretsmanager_secret.jwt_secret.id
  secret_string = var.jwt_secret
}
