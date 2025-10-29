# IAM Role
resource "aws_iam_role" "lambda-role" {

    name                  = "${var.function_name}-lambda-role"
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
# Lambda Policy
resource "aws_iam_policy" "lambda-policy" {

    name    = "${var.function_name}-lambda-policy"
    policy  = jsonencode({
        "Version" : "2012-10-17",
        "Statement" : [
            {
            "Effect" : "Allow",
            "Action" : var.actions,
            "Resource" : var.resources
            }
        ]
    })

}
# Role Policy Attachment
resource "aws_iam_role_policy_attachment" "role-policy" {

    role          = aws_iam_role.lambda-role.name
    policy_arn    = aws_iam_policy.lambda-policy.arn

}
# Source Code .zip Directory
locals {
    source_code_zip_dir = "../api/${var.zip_dir_slice}/deploy/bootstrap.zip"
}
# Lambda Function
resource "aws_lambda_function" "lambda" {

    function_name       = var.function_name
    role                = aws_iam_role.lambda-role.arn
    handler             = "main"
    timeout             = 3
    filename            = local.source_code_zip_dir
    source_code_hash    = filebase64sha256(local.source_code_zip_dir)
    runtime             = "provided.al2"
    architectures       = ["arm64"]
    memory_size         = 128

}