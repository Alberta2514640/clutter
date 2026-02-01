/*
terraform {
  backend "s3" {
    bucket         = "tfstate-us-west-2-5727f4ce"
    key            = "clutter/infra/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "tflock"
    encrypt        = true
  }
}

*/

terraform {
  backend "s3" {
    bucket         = "clutter-tfstate-hamza-amar"  # New bucket in new account
    key            = "clutter/infra/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "tflock-new"  # New DynamoDB table
    encrypt        = true
    profile        = "hamza-amar"
  }
}