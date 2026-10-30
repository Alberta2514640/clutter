# NOW

Updated 2026-10-30.

## Goal

Build System B, the adapter layer, beside System A (Clutter as built, AWS only), so a diagram can deploy to other targets through connectors: General first, then PatriCloud and TierHive. Design: `docs/architecture/provider-adapter.md`. What exists: `docs/architecture/system.md`. Decisions: ADR-001 to ADR-008.

## Where it stands

- The harness, the docs and the design are on branch `design/setup` of `origin` (`FayzanT/re-clutter`, a fork of the client's `Alberta2514640/clutter`, added as remote `upstream`). The first commit, `7b49222`, is pushed; the 2026-10-30 revisions are in the working tree.
- The design now follows the team's plan (`research_notes/Re-Clutter Architecture/`): Option 1, build B next to A and touch A only at named seams (ADR-006); B1 first, B2 later (ADR-007); the General connector first (ADR-008).
- Neither PatriCloud nor TierHive has a public API today. Patrii: REST API and Terraform provider "coming soon" (`docs/research/2026-10-26-patrii-cloud.md`). TierHive: an alpha NAT VPS host whose API ships with its beta, "later in 2026 or early 2027" (`docs/research/2026-10-30-tierhive.md`). Both connectors are blocked; General is not.
- Every local check was run on 2026-10-30 (`docs/architecture/system.md` § Running it). Go build and vet, frontend types, lint and build, Terraform validate and the database schema pass. Go tests fail in the 2 known packages, and `terraform fmt` fails on 7 files.
- The capstone AWS deployment is gone (the API host is NXDOMAIN). A live run needs our own AWS account, Postgres and Google OAuth client; no extra tools.
- All 24 `upstream` branches are merged into `main` or stale, except `dev` (CI `actions/checkout` v6) and `database-seed` (an abandoned DynamoDB seed).
- Waiting on: the team's answers to the questions below, and contacts at Patrii and TierHive.

## Next

Phase 0 (make the seams safe), in this order:

1. Authenticate `POST /terraform-engine` with a signed webhook secret, and decide whether `GET /resources` stays public.
2. Turn on row-level security (or turn off the Supabase Data API) and fix the `anon` policy on `supported_resources`.
3. Validate and escape every canvas value before it reaches a template (allow-lists for enumerated fields, a name pattern, HCL escaping for free text including `${` and `%{`), with rejection tests.
4. Set `GOOGLE_CLIENT_ID` on the log-in Lambda and check `email_verified`.
5. Ownership checks on every child id, swept across all handlers through one shared helper (recent-activity, terraform-command-runner, code-upload, logs/live, ansible list-jobs).
6. Go tests green offline (build-tag the live-S3 test, fix the nil connection panic) and Go build, vet and test in CI.

Then phase 1: the connector interface, the B1 skeleton and the General connector.

## Open defects in A

Verified at the code by the lead:

- `POST /terraform-engine` has no authorizer (`backend/infra/main.tf`, terraform-engine-create integration) and trusts `project_id` from the body. Anyone with a project id can overwrite that tenant's generated code. Critical.
- Row-level security: none of the ten tables in `application-tables.sql` enables it, and `supported_resources` enables it with a policy for `anon` that has no command clause, so it allows every command. On Supabase with the Data API on, the anon key can read users and linked roles and rewrite the resource catalog. The final report claims RLS isolates tenants. High, depending on the Supabase project's settings.
- `GET /resources` has no authorizer. Low impact (static catalogue), but undocumented.
- Customer role grants `ec2:*`, `s3:*`, `iam:*`, `ssm:*` (`client_side_terraform_deployer_role.yaml`). High.
- Lambda runtimes: config panel sends `nodejs24.x`, code upload accepts `nodejs24`. Node code uploads fail.
- Customer state is a local file synced to S3 without a lock; generated code and state sit in `us-west-2` for every customer.
- ECR repos are immutable but images are pushed as `:latest`; the second push fails.
- `build_linux.sh` omits `terraform-engine/code-upload` and `terraform-engine/logs/recent-activity` (`build.sh` lists all 33).
- The frontend members API is an in-memory mock (`MOCK_MEMBERS`); no invite route exists.
- The frontend `RunAction` type offers `plan`; the backend accepts only `apply` and `destroy`.
- Nothing in the repo calls `POST /terraform-engine`; it depends on a Supabase webhook configured outside the repo.
- The frontend proxy lets any `clutter_token` cookie value through; it never verifies the token. Low impact (page shells only, data needs a real JWT).
- `frontend/clutter/init.bash` is a stale scaffold script that overwrites real source files, and the frontend README tells people to run it.
- The Google OAuth client id is hard-coded in two frontend files and belongs to the capstone team's Google project.

Found or confirmed by the 2026-10-30 code sweep (a reader lane; the first four re-checked at the source by the lead):

- Log-in accepts a Google ID token minted for any app. `GOOGLE_CLIENT_ID` is read by the log-in handler but never set on its Lambda (`backend/infra/main.tf`, log-in module), and Google's `idtoken.Validate` skips the audience check when the audience is empty. `email_verified` is not checked. High.
- `GET /terraform-engine/logs/recent-activity` checks membership of the org, then queries by diagram id alone: a cross-tenant read of another tenant's diagram name and deploy history. High.
- Ansible polling never stops on success: the frontend's terminal set is `SUCCESS`, `FAILED`, `CANCELLED`; the database uses `COMPLETED`.
- No deploy sends a region: the account-link page offers a few regions, but `runTerraform` sends none, so every deploy runs in `us-west-2`. The `us-east-1` default is in the API Gateway JSON model, which API Gateway does not apply.
- Template injection is worse than quote-breaking: values sit inside HCL strings, so `${...}` in a canvas value is evaluated by Terraform. Only the Terraform identifier is sanitized; every other value (function name, handler, runtime, instance type, key name, env vars and others) is inserted raw through `text/template`, and the entry point is unauthenticated. Critical. A fix must escape `${` and `%{` as well as `"` and `\`.
- `submit-role-arn` stores the ARN with no STS check; only a regex and the 8-character suffix are checked. Confirmed.
- `terraform-command-runner` and `code-upload` check membership but not that the project and diagram belong to the org (the runner writes a log row on any diagram); `logs/live` describes any task ARN in the shared cluster. Confirmed.
- `ansible/list-jobs` filters by creator only, with no org check. `role_arn` is in the response code but never written. Partly confirmed.
- JWT: no iss or aud, and validation is turned off and done by hand; the log-in handler type-asserts Google claims without `ok` and panics if one is missing. The missing empty-secret check fails closed (the library rejects an empty key). Partly confirmed.
- Any member can delete an organization (no role column).
- Canvas drift: the Lambda `memory_size` choice is ignored (the UI stores strings, the generator reads numbers, so it is always 128); the code-upload handler is never saved, so Node and Python functions keep `handler = "main"`; the API Gateway template is invalid with CORS off; IAM references `aws_apigatewayv2_api` but the template creates a REST API; seed runtime `provided.al2` vs generator `provided.al2023`.
- Frontend: `/register` posts to a dead hard-coded host and is not reachable through the proxy; `settings/organization/create-org` and the workspaces page are stubs; the diagram editor uses the user's first organization, so deploy and upload break for users in more than one.
- Dead code in the generator (`LambdaDefaults`, `S3Defaults`, `DynamoDBDefaults`, `LoadResourceTemplates`, `RenderContext`, `HasDataChanged`). Confirmed.
- Minor: the recent-activity Lambda's log permission points at another log group; the terraform branch of `ansible/run-task` is dead; `application-tables.sql` fails on a re-run (`CREATE TYPE`).

Closed: the audit suspected AWS keys were committed in `ac90930`. That commit added `.vscode/.bashrc` and a `.gitignore` entry only; no AWS key pattern appears in the tree or any commit.

## Open questions

1. What does General (custom) mean: Terraform with a named provider plus existing hosts over SSH (default), or something else?
2. Why TierHive, and does the client have a relationship with KernelKeepers LTD? Default: treat TierHive as a later connector and reach it through General until its API ships.
3. Who contacts Patrii and TierHive, and can we join Patrii's Technology partner program for a sandbox? Default: the client (Alberta 2514640 Inc) makes the contact and asks.
4. Does B run as a container (default, since B1 is self-hosted) or as Lambdas like A?
5. Is Patrii a customer target, a host for Clutter itself, or both? Default: a customer target first.
6. Does a diagram target exactly one target instance? Default: yes.
7. Should our fork pick up `upstream/dev`'s CI fix (`actions/checkout` v6)? Default: yes, in phase 0's CI work.
