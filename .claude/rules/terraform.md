---
paths:
  - "backend/infra/**"
  - "backend/tfstate-infra/**"
  - "**/*.tf"
  - "**/*.tf.tmpl"
---

# Terraform rules (path-scoped)

There are two kinds of Terraform here, and they must never be confused.

- **Platform Terraform** (`backend/infra`, `backend/tfstate-infra`) deploys Clutter itself into Clutter's own AWS account. It is applied by a person, never from a session; the `guard-cloud-writes` hook asks before any `apply`, `destroy` or state edit.
- **Customer templates** (`*.tf.tmpl` under `backend/infra/modules/s3/uploads/<provider>/<service>/`) are Go `text/template` files that the generator renders into the code Clutter runs inside a customer's account. They are uploaded to S3 by the platform Terraform, so a template change ships only on the next platform apply.

Rules:
- Run `terraform fmt` and `terraform validate` on platform modules you change (`terraform init -backend=false` first in a scratch copy if needed). Never run `plan` against the real backend from a session; it reads state that holds secrets.
- A template is checked by rendering it: run the generator tests in `backend/api/terraform-engine/create` and read the rendered `.tf`. `terraform fmt` cannot parse template syntax.
- Every value a user typed that reaches a template is escaped or validated before it is rendered. A diagram name, a tag or a bucket name in a template is an injection path into code that runs with the customer's credentials.
- A template and the Go config struct that feeds it change together, with the frontend node config and the JSON schema in `backend/infra/modules/templates/api-models/schemas/` when the field is user-facing.
- Pin provider versions. Check the resource on the Terraform Registry for the pinned version before adding an argument (`ground-before-building`).
