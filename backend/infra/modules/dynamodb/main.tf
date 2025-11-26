# Clutter - Single-table DynamoDB design
/*
Stores:
USER,
ORGANIZATION,
PROJECT,
DIAGRAMS,
RESOURCE,
CONNECTION
*/
resource "aws_dynamodb_table" "application-data" {

  name         = "application-data"
  billing_mode = "PROVISIONED"
  hash_key     = "PK"
  range_key    = "SK"

  read_capacity  = 1
  write_capacity = 1

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

}

resource "aws_dynamodb_table" "supported-resources-metadata" {

  name         = "supported-resources-metadata"
  billing_mode = "PROVISIONED"
  hash_key     = "PK"
  range_key    = "SK"

  read_capacity  = 1
  write_capacity = 1

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

}
