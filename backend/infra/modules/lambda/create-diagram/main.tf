# Create Diagram IAM Role
resource "aws_iam_role" "create-diagram-lambda-role" {

    name                  = "create-diagram-lambda-role"
    assume_role_policy    = jsonencode({
        "Version": "2012-10-17",
        "Statement": [{
            "Action": "sts:AssumeRole",
            "Effect": "Allow",
            "Sid": "",
            "Principal": {
                "Service": "lambda.amazonaws.com"
            }
        }
    ]
    })

}
# Create Diagram Lambda Policy
resource "aws_iam_policy" "create-diagram-lambda-policy" {

    name    = "create-diagram-lambda-policy"
    policy  = jsonencode({
        "Version" : "2012-10-17",
        "Statement" : [
            {
            "Effect" : "Allow",
            "Action" : ["dynamodb:PutItem"],
            "Resource" : ["*"]
            }
        ]
    })

}
# Create Diagram Role Policy Attachment
resource "aws_iam_role_policy_attachment" "attach-lambda-role-policy" {

    role          = aws_iam_role.create-diagram-lambda-role.name
    policy_arn    = aws_iam_policy.create-diagram-lambda-policy.arn

}
# Generic directories
variable "create-diagram-zip-dir" {

    description = "The directory for the diagram creation Lambda binary zip"
    type        = string
    default     = "../api/diagram/create/deploy/bootstrap.zip"

}
# Lambda: create-diagram
resource "aws_lambda_function" "create-diagram" {

    function_name       = "create-diagram"
    role                = aws_iam_role.create-diagram-lambda-role.arn
    handler             = "main"
    timeout             = 3
    filename            = var.create-diagram-zip-dir
    source_code_hash    = filebase64sha256(var.create-diagram-zip-dir)
    runtime             = "provided.al2"
    architectures       = ["arm64"]
    memory_size         = 128

}