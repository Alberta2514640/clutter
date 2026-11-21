# The platform's state lives in S3. Every setting comes from the environment's backend file at init
# (prod.backend.hcl, copied from prod.backend.hcl.example); the deploy command passes it.
terraform {
  backend "s3" {}
}
