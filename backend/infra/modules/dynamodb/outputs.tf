output "application_data_table_arn" {
  value = aws_dynamodb_table.application-data.arn
}

output "supported_resurces_metadata_table_arn" {
  value = aws_dynamodb_table.supported-resources-metadata.arn
}
