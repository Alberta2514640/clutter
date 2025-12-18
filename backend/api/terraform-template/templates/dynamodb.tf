# DynamoDB Table Template
# Variables: {table_name}, {hash_key}, {range_key}

resource "aws_dynamodb_table" "table" {
  name         = var.table_name
  billing_mode = "PROVISIONED"
  hash_key     = var.hash_key
  range_key    = var.range_key

  read_capacity  = 1
  write_capacity = 1

  attribute {
    name = var.hash_key
    type = "S"
  }

  attribute {
    name = var.range_key
    type = "S"
  }

  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"
}
