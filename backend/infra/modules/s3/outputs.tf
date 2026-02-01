output "bucket_arn" {
  value = aws_s3_bucket.clutter_bucket.arn
}

output "bucket_name" {
  description = "The name of the clutter S3 bucket"
  value       = aws_s3_bucket.clutter_bucket.id
}

output "bucket_id" {
  description = "The ID of the clutter S3 bucket"
  value       = aws_s3_bucket.clutter_bucket.id
}
