variable "function_name" {
  type        = string
  description = "The name of the AWS Lambda function to be created."
}
variable "actions" {
  type        = list(string)
  description = "A list of IAM actions that the Lambda function's policy will allow."
}
variable "resources" {
  type        = list(string)
  description = "A list of AWS resource ARNs that the Lambda function's policy can access."
}
variable "zip_dir_slice" {
  type        = string
  description = "The relative directory path segment used to locate the Lambda deployment package (bootstrap.zip)."
}
variable "environment_variables" {
  type        = map(string)
  description = "A map of environment variables to set for the Lambda function."
  default     = {}
}
