output "table_name" {
  value = aws_dynamodb_table.iac_designer.name
}

output "table_arn" {
  value = aws_dynamodb_table.iac_designer.arn
}

output "table_id" {
  value = aws_dynamodb_table.iac_designer.id
}

output "stream_arn" {
  value = aws_dynamodb_table.iac_designer.stream_arn
}
