# Clutter - Single-table DynamoDB design
# Stores: TENANT, USER, MEMBERSHIP, PROJECT, CANVAS, NODE, EDGE, WORKSPACE, VAR_SET, MODULE_SET, RUN, ACCOUNT_LINK
resource "aws_dynamodb_table" "clutter_ddb" {
  name         = var.table_name
  billing_mode = var.billing_mode
  hash_key     = "pk"
  range_key    = "sk"

  read_capacity  = var.billing_mode == "PROVISIONED" ? var.read_capacity : null
  write_capacity = var.billing_mode == "PROVISIONED" ? var.write_capacity : null

  attribute {
    name = "pk"
    type = "S"
  }

  attribute {
    name = "sk"
    type = "S"
  }

  # GSI1: Tenant → Projects
  attribute {
    name = "gsi1pk"
    type = "S"
  }

  attribute {
    name = "gsi1sk"
    type = "S"
  }

  # GSI2: Canvas → Nodes/Edges
  attribute {
    name = "gsi2pk"
    type = "S"
  }

  attribute {
    name = "gsi2sk"
    type = "S"
  }

  # GSI3: Tenant → Workspaces
  attribute {
    name = "gsi3pk"
    type = "S"
  }

  attribute {
    name = "gsi3sk"
    type = "S"
  }

  # GSI4: VarSets by Environment
  attribute {
    name = "gsi4pk"
    type = "S"
  }

  attribute {
    name = "gsi4sk"
    type = "S"
  }

  # GSI5: Reverse Edge Lookup
  attribute {
    name = "gsi5pk"
    type = "S"
  }

  attribute {
    name = "gsi5sk"
    type = "S"
  }

  # GSI6: Workspace Run History
  attribute {
    name = "gsi6pk"
    type = "S"
  }

  attribute {
    name = "gsi6sk"
    type = "S"
  }

  # GSI7: User Activity
  attribute {
    name = "gsi7pk"
    type = "S"
  }

  attribute {
    name = "gsi7sk"
    type = "S"
  }

  # GSI8: Account Lookup
  attribute {
    name = "gsi8pk"
    type = "S"
  }

  attribute {
    name = "gsi8sk"
    type = "S"
  }

  # Global Secondary Indexes (8 total)
  dynamic "global_secondary_index" {
    for_each = [
      { name = "GSI1", hash = "gsi1pk", range = "gsi1sk" },
      { name = "GSI2", hash = "gsi2pk", range = "gsi2sk" },
      { name = "GSI3", hash = "gsi3pk", range = "gsi3sk" },
      { name = "GSI4", hash = "gsi4pk", range = "gsi4sk" },
      { name = "GSI5", hash = "gsi5pk", range = "gsi5sk" },
      { name = "GSI6", hash = "gsi6pk", range = "gsi6sk" },
      { name = "GSI7", hash = "gsi7pk", range = "gsi7sk" },
      { name = "GSI8", hash = "gsi8pk", range = "gsi8sk" },
    ]

    content {
      name            = global_secondary_index.value.name
      hash_key        = global_secondary_index.value.hash
      range_key       = global_secondary_index.value.range
      projection_type = "ALL"
      read_capacity   = var.billing_mode == "PROVISIONED" ? var.gsi_read_capacity : null
      write_capacity  = var.billing_mode == "PROVISIONED" ? var.gsi_write_capacity : null
    }
  }

  # TTL for automatic item expiration
  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  # Point-in-time recovery
  point_in_time_recovery {
    enabled = var.enable_point_in_time_recovery
  }

  # Encryption at rest
  server_side_encryption {
    enabled     = true
    kms_key_arn = var.kms_key_arn
  }

  # DynamoDB Streams for change data capture
  stream_enabled   = var.enable_streams
  stream_view_type = var.enable_streams ? "NEW_AND_OLD_IMAGES" : null
}