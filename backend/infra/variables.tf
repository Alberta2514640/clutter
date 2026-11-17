variable "aws_region" {
  type        = string
  description = "The AWS region to deploy resources to"
}
variable "jwt_secret" {
    type        = string
    description = "JWT secret key"
}
variable "psql_connection_string" {
    type        = string
    description = "PostgreSQL connection string to enable querying"
}
variable "clutter_account_id" {
  description = "AWS Account ID for Clutter Fargate deployment role"
  type        = string
}
variable "site_domain" {
  description = "The site's one DNS name, such as app.example.com; https://<site_domain> serves the website and B API (/v1/*), and is the origin A and B answer CORS for"
  type        = string
}
variable "ansible_runner_image_tag" {
  description = "Version tag of A's Ansible runner image, pushed by modules/fargate/ecr-scripts/ansible-runner/build-and-push.sh; never latest"
  type        = string
}
variable "terraform_deployer_image_tag" {
  description = "Version tag of A's Terraform deployer image, pushed by modules/fargate/ecr-scripts/terraform-deployer/build-and-push.sh; never latest"
  type        = string
}
variable "google_client_id" {
  description = "Clutter's Google OAuth client id; log-in refuses tokens minted for any other app"
  type        = string
}
variable "contact_email" {
  description = "The address the website's public pages give for contact (footer, FAQ, privacy policy, terms); deploy.sh --site-image builds the site with it. Read by no resource"
  type        = string
}
variable "adapter_image_tag" {
  description = "Version tag of B's images (clutter-adapter-api and clutter-adapter-runtask), pushed by backend/adapter/images/build-and-push.sh; never latest"
  type        = string
}
variable "certificate_arn" {
  description = "ARN of the ACM certificate (in aws_region) for site_domain"
  type        = string
}
variable "site_image_tag" {
  description = "Version tag of the website's image (clutter-website), pushed by npm run deploy -- prod --site-image; never latest"
  type        = string
}
