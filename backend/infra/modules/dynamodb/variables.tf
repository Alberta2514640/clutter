variable "table_name" {
  type        = string
  description = "Name of the DynamoDB table"
  default     = "IaCDesigner"
}

variable "billing_mode" {
  type        = string
  description = "Billing mode for the table"
  default     = "PROVISIONED"
}

variable "read_capacity" {
  type        = number
  description = "Read capacity units (only used if billing_mode is PROVISIONED)"
  default     = 5
}

variable "write_capacity" {
  type        = number
  description = "Write capacity units (only used if billing_mode is PROVISIONED)"
  default     = 5
}

variable "gsi_read_capacity" {
  type        = number
  description = "Read capacity units for GSIs"
  default     = 5
}

variable "gsi_write_capacity" {
  type        = number
  description = "Write capacity units for GSIs"
  default     = 5
}

variable "enable_point_in_time_recovery" {
  type        = bool
  description = "Enable point-in-time recovery"
  default     = true
}

variable "enable_streams" {
  type        = bool
  description = "Enable DynamoDB Streams"
  default     = false
}

variable "kms_key_arn" {
  type        = string
  description = "KMS key ARN for encryption"
  default     = null
}

variable "environment" {
  type        = string
  description = "Environment name"
  default     = "dev"
}

variable "tags" {
  type        = map(string)
  description = "Additional tags"
  default     = {}
}
