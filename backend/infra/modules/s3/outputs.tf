output "clutter_bucket_arn" {
  description = "ARN of the Clutter S3 bucket"
  value       = aws_s3_bucket.clutter_bucket.arn
}

output "clutter_bucket_name" {
  description = "Name of the Clutter S3 bucket"
  value       = aws_s3_bucket.clutter_bucket.bucket
}
output "clutter_bucket_name" {
  value = aws_s3_bucket.clutter_bucket.bucket
}