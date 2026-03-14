terraform {
  backend "s3" {
    bucket         = "tfstate-us-west-2-fc56b463"
    key            = "clutter/infra/terraform.tfstate"
    region         = "us-west-2"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
  }
}
