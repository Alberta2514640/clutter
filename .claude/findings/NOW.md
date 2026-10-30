# NOW

Updated 2026-10-26.

## Goal

Build the provider adapter layer so Clutter can target clouds beyond AWS, starting with Patrii Cloud. Design: `docs/architecture/provider-adapter.md`. Research: `docs/research/2026-10-26-patrii-cloud.md`.

## Where it stands

- The Claude harness is set up (root and folder `CLAUDE.md`, `.claude/` agents, skills, rules, hooks). Nothing in the product code has changed.
- The design now has two halves: an inbound `/v1` standard API (ADR-005) and outbound target adapters. How to reach Patrii (OpenStack and Terraform vs Patrii's console REST API) is contested (ADR-003).
- `reports/` and `research_notes/` hold a second research run written outside this session on 2026-10-26. They are untracked; decide whether to commit them or fold them into `docs/research/`.
- Patrii publishes no documented customer API yet (REST API and Terraform provider both "coming soon"). Phase 3 is blocked on Patrii's answer; phases 0 to 2 are not.
- Waiting on: the team's answers to the open questions below, and a contact at Patrii.

## Next

Phase 0 (make the base safe), in this order:
1. Authenticate `POST /terraform-engine` with a signed webhook secret, and decide whether `GET /resources` stays public.
2. Validate and escape every canvas value before it reaches a template (allow-lists for enumerated fields, a name pattern, HCL escaping for free text), with rejection tests.
3. Ownership checks on every child id, swept across all handlers through one shared helper.
4. Go tests green offline (build-tag the live-S3 test, fix the nil connection panic) and Go build, vet and test in CI.

## Open defects (2026-10-26 audit)

Verified at the code by the lead:
- `POST /terraform-engine` has no authorizer (`backend/infra/main.tf`, terraform-engine-create integration) and trusts `project_id` from the body. Anyone with a project id can overwrite that tenant's generated code. Critical.
- `GET /resources` has no authorizer. Low impact (static catalogue), but undocumented.
- Customer role grants `ec2:*`, `s3:*`, `iam:*`, `ssm:*` (`client_side_terraform_deployer_role.yaml`). High.
- Templates insert canvas values raw into HCL strings (`{{.ResourceName}}`, `{{.InstanceType}}` and others). Whether the Go side validates them first is not yet checked; the audit says it does not. Potentially critical (code execution in the deployer, which holds the Postgres connection string and a cross-account role).
- Lambda runtimes: config panel sends `nodejs24.x`, code upload accepts `nodejs24`. Node code uploads fail.
- Customer state is a local file synced to S3 without a lock; generated code and state sit in `us-west-2` for every customer.
- ECR repos are immutable but images are pushed as `:latest`; the second push fails.
- `build_linux.sh` omits `terraform-engine/code-upload` and `terraform-engine/logs/recent-activity`.
- The frontend members API is an in-memory mock (`MOCK_MEMBERS`); no invite route exists.
- The frontend `RunAction` type offers `plan`; the backend accepts only `apply` and `destroy`.
- Nothing in the repo calls `POST /terraform-engine`; it depends on a Supabase webhook configured outside the repo.

Reported by the audit, not yet verified by the lead:
- `submit-role-arn` stores the ARN without an STS check.
- `terraform-command-runner` and `code-upload` check org membership but not that the project and diagram belong to the org; `logs/live` takes any task ARN.
- JWT: no empty-secret check, no iss or aud; the login handler type-asserts Google claims without `ok` and panics if `picture` is missing.
- Drift: schema region default `us-east-1` vs code `us-west-2`; frontend offers 2 regions vs 17; seed Lambda runtime `provided.al2` vs generator `provided.al2023`.
- Dead code in the generator (`LambdaDefaults`, `S3Defaults`, `DynamoDBDefaults`, `LoadResourceTemplates`, `RenderContext`, `HasDataChanged`).

Closed: the audit suspected AWS keys were committed in `ac90930`. That commit added `.vscode/.bashrc` and a `.gitignore` entry only; no AWS key pattern appears in the tree or any commit.

## Open questions

1. Is Patrii a customer target, a hosting target for Clutter itself, or both? Default: customer target first.
2. Does a diagram target one provider only? Default: yes.
3. Who talks to Patrii, and can we join their Technology partner program for a sandbox? Default: the client (Alberta 2514640 Inc) makes the contact.
4. Is this repo continuing as the team's shared repo, with the harness committed for everyone? Default: yes, on a branch for review.
