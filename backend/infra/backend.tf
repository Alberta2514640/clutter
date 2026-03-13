# terraform {
#   backend "s3" {
#     bucket         = "tfstate-us-west-2-5727f4ce"
#     key            = "clutter/infra/terraform.tfstate"
#     region         = "us-west-2"
#     dynamodb_table = "tflock"
#     encrypt        = true
#   }
# }

terraform {
  backend "s3" {
    bucket         = "tfstate-us-west-2-7341aab7"
    key            = "clutter/infra/developer/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "tflock"
    encrypt        = true
  }
}
