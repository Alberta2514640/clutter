# System A: Clutter as built (exists today)

As it stands on 2026-11-14 on the `impl/b0` branch, from a read of the code and a local run of every route (`backend/devserver`). The client's `main` does not have the stage 1 fixes yet. The final report (`docs/old-capstone/final-report.pdf`, same text in `.md`) describes the design as submitted; where they differ, this file and the code win. What is planned next is `system-b-adapter-layer.md`.

## Parts

- **Frontend** (`frontend/clutter`): Next.js 16 App Router, React 19, React Flow (`@xyflow/react`) for the canvas, TanStack Query for server data, Zustand for UI state, Tailwind 4. Google sign-in. In the cloud it runs as its own Fargate service behind B's load balancer (ADR-023), from an image the deploy command builds and pushes to Clutter's ECR.
- **API** (`backend/api`): Go Lambdas, one per route, behind AWS API Gateway. Shared code in `generic` (Postgres access, membership and ownership checks, responses, regions). Postgres (Supabase) holds users, orgs, projects, diagrams, supported resources, linked accounts, runs and jobs.
- **Platform infrastructure** (`backend/infra`): Terraform for API Gateway, the Lambdas, S3, ECR, ECS Fargate and SQS, in Clutter's own AWS account, with remote state in S3 and a DynamoDB lock (`backend/infra-state-bootstrap` bootstraps that).
- **Runners** (`backend/infra/modules/fargate/ecr-scripts`): two container images run as Fargate tasks. `terraform-deployer` runs Terraform for a diagram; `ansible-runner` runs a playbook on the customer's EC2 instances.

## The two worlds

This section describes A's case, where world 2 is always an AWS account. The names for every cloud are defined in `system-b-adapter-layer.md` § Names.

A is a hosted service (SaaS): one copy of Clutter, run for every user. Clutter touches two separate cloud accounts. **World 1** is Clutter's own account: the website, the API, the database and the workers. **World 2** is the user's own account, where the things they draw get built. The user owns and pays for world 2; Clutter only builds there, with access the user grants. A user's diagram is data in world 1; what it builds lives in world 2.

| | World 1: Clutter's account | World 2: the user's account |
| --- | --- | --- |
| Holds | the website, the Lambdas and API Gateway, the Fargate workers | whatever the user drew: Lambda, S3, DynamoDB, API Gateway, EC2 |
| Postgres (Supabase) | users, organizations, members, projects, diagrams and history, linked accounts, deploy logs, Ansible jobs, playbook records, the resource catalog | none from Clutter |
| S3 | templates; generated Terraform and `status.json`; Terraform state; deploy and job logs; uploaded Lambda code and playbooks | only buckets the user drew |
| Credentials | the JWT secret and the database password; for each linked AWS account, only a role ARN and an external id | the role the user created for Clutter |

How the worlds meet: the user creates a role in their account through a CloudFormation link and pastes its ARN into Clutter. At deploy, a Fargate worker in world 1 asks AWS (STS) for short-lived keys for that role, then runs Terraform, which builds in world 2. Ansible runs the same way and configures the user's EC2 servers through SSM. Clutter never stores a permanent key for world 2.

Terraform state for every user's diagram sits in world 1, in Clutter's bucket in the region Clutter runs in (`ca-central-1` in the cloud), with no lock. It lists everything in the user's account and can hold secrets (an open defect in `.claude/findings/NOW.md`).

An EC2 server is the user's, not Clutter's: Clutter creates it with Amazon Linux 2023, the chosen instance type, an SSM role and an optional SSH key, and nothing else installed. What runs on it comes from the user's own Ansible playbook. Its CPU and memory come from the instance type; its disk is the image's default, with no size option; Clutter sets no security group.

## How A runs: there is no backend server

A has no server of its own, and it never did. The capstone team built it to run only in AWS: each route is a small function (a Lambda) that AWS starts on demand, and API Gateway gives those functions their URLs. There was nothing to start on a laptop, no `go run`, and no local mode. They developed against their AWS account.

The app's code is the same everywhere. To run it on a laptop, we added stand-ins for the AWS parts around it, all in `backend/devserver`. That extension is **A2**: it is not part of A and never ships to end users. A's code does not know about it, with one exception: the **Link fake account** button, which appears only in a local development build.

| In real AWS (System A) | Stand-in on your laptop (ours) |
| --- | --- |
| API Gateway gives the functions URLs | the gateway on port 8787 |
| Lambda service runs the functions | the same Go code, run as local processes |
| Lambda authorizer checks the token | the same authorizer code |
| Google sign-in | the same Google sign-in |
| Supabase Postgres | Postgres in Docker |
| S3 holds templates, generated files and logs | a local S3, on disk in `backend/devserver/.state/s3` |
| Frontend on a server, built with the API address | `npm run dev` on port 3100, behind the front door on port 3000 |
| Fargate deployer runs Terraform in the customer's account | nothing: Deploy fails locally |
| SQS and Fargate runner for Ansible | nothing: jobs stop at SQS |

## What changed since the capstone

The capstone's version is tagged `capstone-original`. This is what is different now.

| Area | Capstone delivered | Now |
| --- | --- | --- |
| Where it runs | only in AWS; that account is gone | locally with `npm run dev` (A2); the cloud environment is not set up yet |
| Reading generated Terraform | not in the app, only through the AWS console | **Preview Terraform** in the editor, for every user (ADR-012) |
| Generator | writes `.tf` files | also records `status.json`, and removes a stale `outputs.tf` |
| What triggers generation | a Supabase database webhook configured outside the repo, on a route with no authorizer | the editor calls `POST /terraform-engine` with the user's token after each save; the handler checks ownership and reads the diagram from the database (ADR-018) |
| Ownership checks | five handlers checked only membership in the organization | every handler that takes a project, diagram or run id proves it belongs to the caller's organization through one shared helper in `generic` |
| Canvas values in templates | inserted raw, so `${...}` in a value ran inside Terraform | names checked against one name rule (`generic/hclsafe`), enumerated fields against an allow-list beside each generator (Lambda runtimes in `generic.LambdaRuntimes`), and free text escaped for HCL, including `${` and `%{` |
| Row-level security | on for `supported_resources` only, with a policy that let the public `anon` role run every command | on for every table; `anon` may only read the resource catalog |
| Google sign-in | `GOOGLE_CLIENT_ID` unset, so a token minted for any app passed; `email_verified` unchecked; a missing claim panicked | the audience is checked, an unverified email is refused, and every claim is read with a type check |
| Sign-in check | inside the authorizer's own package | `generic.VerifyUserJWT`, shared by A's authorizer and B |
| Customer role | `ec2:*`, `s3:*`, `iam:*`, `ssm:*` and more | named actions only; changes only resources tagged `ManagedBy`; roles and users only under `/clutter/` with a permissions boundary (`RoleTemplateVersion` 8). Real AWS must confirm it |
| Runners | both handed the database string | neither holds it; each writes a status file to S3, and the API records it after its ownership checks. Real AWS must confirm the S3 writes |
| Region | the generator's fixed `us-west-2` | the environment's region (`generic.PlatformRegion`; `ca-central-1` in the cloud) |
| Template loading | inside the generator's `internal` package | `generic/tftemplate`, shared with B's AWS connector; a golden test keeps A's output byte-identical |
| IAM from edges | "reads" edges also granted writes; the API Gateway case named an API type no template renders | reads only, as an inline policy on the function's role; the invoke permission names the REST API |
| Generation errors | an output template failure was hidden; error bodies carried paths and SDK text; API Gateway with CORS off rendered invalid Terraform | a failed output template fails the run; error bodies are fixed sentences; CORS off renders valid Terraform |
| API | 31 routes | 32: adds `GET /terraform-engine/files` |
| Unlocking the canvas locally | not possible | a local-only **Link fake account** button |
| Tests before merging | none on Go | `checks.yml` on every pull request runs every Go package offline, the frontend's checks and the smoke test of every route |
| Frontend packages | `npm audit` reported 2 critical and 14 high issues | no known vulnerability in a package the app ships |
| The website's shared screens | the capstone's screens, with mock members and stub pages | sign-in, onboarding, organizations (with a switcher), projects, Connections and the landing page reworked as one product (ADR-024); A's editor is unchanged |
| Docs | one report, stale READMEs | one owner per fact (`docs/README.md`) |

Unchanged: the handlers not named above, the schema's tables and columns, and the open defects in `.claude/findings/NOW.md`. The customer templates change only where stage 1 needed it: escaping, the `/clutter/` path and boundary on the roles they create, the `ManagedBy` tag, and the API Gateway fixes.

## The capstone's report

Dropped from the original scope, for time: inviting users to an organization with shared diagramming, and pulling an EC2 environment down to a local machine. The database moved from DynamoDB to Postgres mid-project, and the report's glossary still describes DynamoDB.

Testing in the report is manual: a Postman collection, exploratory UI testing, one real deploy and an Ansible playbook run, and informal usability sessions. It gives no counts or timings. The Go tests in the repo are not mentioned.

The report makes claims the capstone's code did not support:

| Report says | The capstone's code | Now, on `impl/b0` |
| --- | --- | --- |
| Every endpoint requires a JWT | `POST /log-in`, `POST /terraform-engine` and `GET /resources` had no authorizer | `POST /log-in` and `GET /resources` (the static resource catalog) have none, on purpose |
| Supabase row-level security keeps each user's data separate | only `supported_resources` enabled it, with a policy that gave the public `anon` role every command; the other ten tables had none | on for every table (§ What changed) |
| Least-privilege roles | the customer role granted `ec2:*`, `s3:*`, `iam:*`, `ssm:*` and more | named actions, limited by tags and a boundary; real AWS must confirm it |
| A new resource needs only a new template in S3 | a resource lived in five places that must agree | the same (`add-cloud-resource`) |
| Generation runs when the user deploys (one passage) or on every save (another) | a Supabase webhook, configured outside the repo, called it | the editor asks for it after each successful save; the save handlers do not call it (ADR-018) |
| No CI | CI built and pushed the frontend image | CI runs the checks on every pull request and builds the website's image without pushing it |

## What A can and cannot do

Status: **proven** means we ran it locally (2026-11-04). **Unproven** means the code exists but needs AWS, so only the capstone team has run it, and their evidence is one Lambda and one playbook.

| Ability | Status |
| --- | --- |
| Sign in with Google; a user can have many organizations and projects | proven |
| Create, rename and delete organizations, projects and diagrams | proven |
| Link an AWS account (CloudFormation link, then paste the role ARN) | proven locally as far as the database; A checks only the ARN's form, and B proves the role with STS when it becomes a connection (ADR-019) |
| Draw on the canvas: 5 resources, connect them, configure them (config panels come from the `supported_resources` table), save, reload | proven |
| Generate Terraform on save, with IAM from edges | proven; output passes `terraform validate` |
| Read and download generated Terraform in the editor (needs a linked account, like the canvas) | proven |
| Deploy and destroy into the customer's AWS account (Destroy is enabled only after a first deploy, per the report) | unproven |
| Live deploy status | unproven |
| Deploy logs saved and listed | listing proven, real logs unproven |
| Upload Lambda code | upload link proven; the runtimes come from one list (`generic.LambdaRuntimes`), so Node uploads no longer fail |
| Run an Ansible playbook on an EC2 instance | unproven; the instance id is pasted by hand |

Resources and the options each offers:

| Resource | Options | Creates |
| --- | --- | --- |
| Lambda | name, handler, timeout, runtime, architecture, memory (ignored, always 128) | function and its IAM role |
| S3 | name, versioning, block public access | bucket and access block |
| DynamoDB | name, billing mode, hash key and type, TTL | table (no sort key) |
| API Gateway | name, description, stage, CORS, HTTP methods | REST API, methods, deployment and stage |
| EC2 Container | name, instance type, key name | instance and an SSM role |

Edges (the lines a user draws between resources): the canvas refuses anything into API Gateway, anything out of S3 or DynamoDB, and a node to itself. Only three edges do anything: Lambda to DynamoDB and Lambda to S3 (read permissions only, as an inline policy on the function's role), and API Gateway to Lambda (permission to invoke, but no route that sends traffic to the function). Any other allowed edge, such as Lambda to EC2, draws a line and generates nothing, with no warning.

A cannot:

- deploy anywhere but AWS, or to any region but the one Clutter runs in (the editor sends no region)
- create networking (VPCs, subnets, security groups), databases such as RDS, queues, load balancers, DNS, or any resource beyond the five
- set an EC2 server's disk size, or give an EC2 server access to S3 or DynamoDB (an edge from EC2 draws a line and grants nothing)
- set Lambda environment variables or tags from the UI
- preview a change (no plan step), estimate cost, import existing infrastructure, or detect drift
- undo, autosave, or show diagram history (history is written, never shown)
- invite members, set roles, or share a diagram; an organization has only its creator
- be called by another program (no API keys or service accounts)

## Inventory

- **API:** 32 routes, one Lambda each. Two have no authorizer: `POST /log-in` and `GET /resources` (the static resource catalog, public on purpose).
- **Lambdas:** 34 with a `main.go`, all wired in the platform Terraform: the 32 routes, the authorizer, and `ansible/run-task` (an SQS consumer).
- **Database:** 11 tables (10 in `application-tables.sql`, plus `supported_resources` in the seed) and 2 enums, with row-level security on every table. `jobs` has no organization column. B's own tables are in `adapter-tables.sql` (`backend/adapter/README.md`).
- **Frontend:** login, onboarding, the dashboard, projects, organization settings, the diagram editor and the AWS account page call the real API. The members page shows the member count and the signed-in member, because the backend has no member names or invites. The diagram editor calls `POST /terraform-engine` after each save. No page calls `GET /ansible/jobs`, the job logs route or `/user-information`. The website also holds B's screens (`frontend/clutter/README.md` § B's screens).

## The main flows

1. **Sign in.** Google ID token in, user upserted, Clutter issues an HS256 JWT (24 hours). A Lambda authorizer checks it on protected routes. The frontend's route guard only checks that a `clutter_token` cookie exists, not that it is valid; data calls still need a real JWT.
2. **Link an AWS account.** Clutter builds a CloudFormation quick-create link for a role in the customer's account (with an external id), and the customer pastes back the role ARN. The ARN is stored, not verified.
3. **Draw and generate.** After a save succeeds, the editor calls `POST /terraform-engine` with the user's token and only `projectId` and `diagramId`. The handler checks that the caller is a member of the project's organization and that the diagram belongs to the project, then reads the diagram from the database; it never takes diagram data from the request. A failed generation never undoes the save. The generator parses each node's label into a resource type, loads that type's `*.tf.tmpl` from S3, renders it with `text/template`, builds IAM from the diagram's edges, emits a fixed `provider "aws"` block, and writes the files to S3 under the org, project and diagram, in a `terraform/` folder, with a `status.json` recording whether that run succeeded. `GET /terraform-engine/files` reads them for the editor's Preview Terraform panel, after checking the caller belongs to that diagram's organization.
4. **Deploy.** `terraform-command-runner` starts a Fargate task with the role ARN and external id. The deployer syncs the files from S3, assumes the customer role, runs init and apply or destroy, syncs local state back to S3 (no lock), and writes its final status as `status.json` beside the run's logs in S3; the API's log routes record it in Postgres when the editor next asks. Logs come from S3 and CloudWatch.
5. **Configure.** A playbook is uploaded, a job is queued on SQS, a consumer starts the `ansible-runner` task, which assumes the customer role and runs the playbook through SSM Run Command.

## Where AWS is assumed

Everywhere a connector would need to reach. Behind a usable seam: the per-resource generators (`ResourceGenerator` interface) and the templates (keyed by a path prefix). Leaked into the core: the resource type constants and label switch, the emitted provider block, IAM from edges, the region list, the account-link handlers and table, the runner scripts' STS and SSM calls, the seed table (no provider column), and the frontend palette, node component and config panel. Under Option 1 (ADR-006) these stay in A; `system-b-adapter-layer.md` maps each to the connector part that replaces it if AWS later moves into B.

## Load-bearing facts

- Organization membership is a boolean (no role column).
- Run state is split: Terraform runs in `diagram_deployment_logs`, Ansible runs in `jobs`.
- Generated code and Terraform state for every customer sit in Clutter's own bucket, in the region Clutter runs in (`generic.PlatformRegion`).
- Customer Terraform state is a local file synced to S3 per diagram; two concurrent runs on one diagram overwrite each other.
- Neither runner holds the Postgres connection string; each writes a status file to S3 that the API records. Both share one task role that may assume any customer's Clutter role (guarded by the external id, which a run knows only for its own organization) and may read and write the whole Clutter bucket, so one run can reach another customer's state and logs. The Ansible job's SQS message carries the role ARN and external id.
- The schema in `backend/storage-sql` is written for Supabase: the seed creates a policy for the `anon` role, which plain Postgres does not have.

## Repositories and branches

- This clone's `origin` is `FayzanT/re-clutter`, a fork made 2026-10-19 of the client's `Alberta2514640/clutter` (remote `upstream`). Work happens in the fork, and pull requests merge it into the client's repo.
- `upstream` has 24 older branches. Every one is merged into `main` or holds an older copy of a file `main` rewrote later, with two exceptions: `dev` bumps `actions/checkout` to v6 in CI, and `database-seed` holds an abandoned DynamoDB seed script. One old PR is open (#21, conflicting).
- CI: `docker-image.yml` builds the website's image on every pull request and push to `main`, with stand-in values, and pushes nothing. `checks.yml` is described in the root `CLAUDE.md` § Commands.
