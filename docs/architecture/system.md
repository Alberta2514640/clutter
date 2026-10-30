# System architecture

As it stands on 2026-10-26, from a read of the code. The root `README.md` describes the design as it was submitted; where they differ, this file and the code win.

## Parts

- **Frontend** (`frontend/clutter`): Next.js 16 App Router, React 19, React Flow (`@xyflow/react`) for the canvas, TanStack Query for server data, Zustand for UI state, Tailwind 4. Google sign-in. Shipped as a Docker image to GHCR by CI.
- **API** (`backend/api`): Go Lambdas, one per route, behind AWS API Gateway. Shared code in `generic` (Postgres access, membership, responses, regions). Postgres (Supabase) holds users, orgs, projects, diagrams, supported resources, linked accounts, runs and jobs.
- **Platform infrastructure** (`backend/infra`): Terraform for API Gateway, the Lambdas, S3, ECR, ECS Fargate and SQS, in Clutter's own AWS account, with remote state in S3 and a DynamoDB lock (`backend/tfstate-infra` bootstraps that).
- **Runners** (`backend/infra/modules/fargate/ecr-scripts`): two container images run as Fargate tasks. `terraform-deployer` runs Terraform for a diagram; `ansible-runner` runs a playbook on the customer's EC2 instances.

## The main flows

1. **Sign in.** Google ID token in, user upserted, Clutter issues an HS256 JWT (24 hours). A Lambda authorizer checks it on protected routes.
2. **Link an AWS account.** Clutter builds a CloudFormation quick-create link for a role in the customer's account (with an external id), and the customer pastes back the role ARN. The ARN is stored, not verified.
3. **Draw and generate.** `POST /terraform-engine` takes a Supabase database-webhook body. The webhook itself is configured in Supabase, outside this repo, and the frontend never calls the route; the diagram update handler does not call it either. So whether a save compiles depends on configuration this repo does not hold. The generator parses each node's label into a resource type, loads that type's `*.tf.tmpl` from S3, renders it with `text/template`, builds IAM from the diagram's edges, emits a fixed `provider "aws"` block, and writes the files to S3 under the org, project and diagram.
4. **Deploy.** `terraform-command-runner` starts a Fargate task with the role ARN and external id. The deployer syncs the files from S3, assumes the customer role, runs init and apply or destroy, syncs local state back to S3 (no lock), and writes status to Postgres. Logs come from S3 and CloudWatch.
5. **Configure.** A playbook is uploaded, a job is queued on SQS, a consumer starts the `ansible-runner` task, which assumes the customer role and runs the playbook through SSM Run Command.

## Where AWS is assumed

Everywhere a provider adapter would need to reach. Behind a usable seam: the per-resource generators (`ResourceGenerator` interface) and the templates (keyed by a path prefix). Leaked into the core: the resource type constants and label switch, the emitted provider block, IAM from edges, the region list, the account-link handlers and table, the runner scripts' STS and SSM calls, the seed table (no provider column), and the frontend palette, node component and config panel. The adapter design (`provider-adapter.md`) maps each to the method that replaces it.

## Load-bearing facts

- The platform Terraform defines the API Gateway methods but no deployment or stage, so publishing the API happens outside this repo.
- Organization membership is a boolean (no role column). The frontend's member management is an in-memory mock (`MOCK_MEMBERS`); no invite route exists.
- Run state is split: Terraform runs in `diagram_deployment_logs`, Ansible runs in `jobs`.

- Templates are fetched from S3 at run time; the platform Terraform uploads each file individually.
- The template loader and writer use a fixed S3 region (`us-west-2`), so generated code and Terraform state for every customer sit in that US region.
- Customer Terraform state is a local file synced to S3 per diagram; two concurrent runs on one diagram overwrite each other.
- The deployer task holds the Postgres connection string and a role that may assume any customer's Clutter role, guarded only by the external id.
