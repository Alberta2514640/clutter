variable "aws_region" {
  type        = string
  description = "Current AWS region"
}

variable "jwt_authorizer_arn" {
  description = "The ARN of the Lambda function used as the JWT authorizer for API Gateway"
  type        = string
}