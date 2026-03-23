terraform {
  backend "s3" {
    bucket         = "tfstate-us-west-2-7341aab7"
    key            = "clutter/infra/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "tflock"
    encrypt        = true
  }
}
