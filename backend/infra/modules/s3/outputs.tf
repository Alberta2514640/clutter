output "bucket_name" {
  value = aws_s3_bucket.iac_storage.id
}

output "bucket_arn" {
  value = aws_s3_bucket.iac_storage.arn
}
