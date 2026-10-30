# backend/infra

Platform Terraform: deploys Clutter itself into Clutter's AWS account. Rules for Terraform here and for the customer templates under `modules/s3/uploads/` are in `.claude/rules/terraform.md`.

Traps:

- **Applied by a person, never from a session.** The `guard-cloud-writes` hook asks before any apply, destroy or state edit. Never run `plan` against the real backend from a session: it reads state that holds secrets.
- **A Lambda route is public unless its integration module gets `jwt_authorizer_id`.** The integration template sets `authorization = "NONE"` when it is missing.
- **Each customer template is uploaded by its own `aws_s3_object` in `modules/s3/main.tf`.** A new template file needs a new object block.
- **The customer role (`modules/s3/uploads/client_side_terraform_deployer_role.yaml`) is close to admin** (`iam:*`, `ec2:*`, `s3:*`, `ssm:*` on `*`). Narrowing it is planned work; widening it needs a reason in the PR.
- **ECR repositories are immutable, but the deployer image is pushed as `:latest`.** A second push fails until the build script tags by version.
- **A route change here is also a devserver change.** `go -C backend/devserver test ./...` parses `main.tf` and fails until `backend/devserver/internal/routes/routes.go` matches (paths, methods, authorizer, timeouts, environment keys).
- **The AWS provider constraint is `>= 4.0.0` with no upper bound**, and no `required_version` is set. Pin both before the next apply.

`backend/tfstate-infra` bootstraps the S3 bucket and DynamoDB lock table for this folder's own state.
