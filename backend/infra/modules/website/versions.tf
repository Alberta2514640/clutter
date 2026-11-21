# The website's part of the platform (docs/architecture/environments.md). The
# versions are pinned once, in backend/infra/providers.tf.
terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}
