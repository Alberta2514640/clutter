resource "random_id" "suffix" {
  byte_length = 4
}
# Clutter infra tfstate S3 Bucket
resource "aws_s3_bucket" "tfstate-bucket" {
    bucket = "tfstate-${var.aws_region}-${random_id.suffix.hex}"
}
resource "aws_s3_bucket_versioning" "tfstate-bucket-versioning" {
    bucket = aws_s3_bucket.tfstate-bucket.id
    versioning_configuration {
      status = "Enabled"
    }
}
resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate-bucket-encryption" {
    bucket = aws_s3_bucket.tfstate-bucket.id
    rule {
        apply_server_side_encryption_by_default {
            sse_algorithm = "AES256"
        }
    }
}
# Clutter infra tflock DynamoDB
resource "aws_dynamodb_table" "tflock-ddb" {
    name         = "tflock"
    billing_mode = "PAY_PER_REQUEST"
    hash_key     = "LockID"

    attribute {
        name = "LockID"
        type = "S"
    }
}