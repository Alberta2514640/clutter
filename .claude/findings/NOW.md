# NOW

Updated 2026-11-21.

## Goal

Clutter becomes one hosted product (B0) where users link their own clouds (world 2.1) and deploy one diagram to any of them: AWS through A today, and every cloud through B's connectors, OpenStack first (ADR-013, ADR-014). Make A safe first, build B beside it, then run it all in one cloud environment (ADR-015). C is A + A2 + B (ADR-011).

- Design: `docs/architecture/system-b-adapter-layer.md`.
- Build order: `docs/architecture/implementation-plan.md`.
- What exists: `docs/architecture/system-a-as-built.md`.
- Decisions: ADR-001 to ADR-028 in `docs/decisions.md`.
- Environments: `docs/architecture/environments.md`.

## Where it stands

- Work happens in the fork (`origin`) and merges by pull request into the client's repo (`upstream`). The client's `main` stops at PR #96, behind the fork's `main`.
- Stages 1 to 6 (with 5b and 6b) are built and tested locally, AWS on LocalStack and OpenStack on DevStack.
- Clutter on a laptop has deployed every example into a real AWS account and VEXXHOST (`docs/reports/2026-11-20-real-cloud-verification.pdf`). That run confirmed the customer role (ADR-027).
- A's runner fixes still need Clutter hosted in AWS. Stage 7 is written in code and waits for the client's accounts.
- Locally, A runs through `backend/devserver` (ADR-009). Deploy and running Ansible in A need the client's AWS account.
- In flight: the website redesign (ADR-028) on `impl/ux-redesign`, not committed. Its local baseline passes: every example and block field on LocalStack and DevStack, and every check (`frontend/clutter/README.md` § Last browser check).
- The redesigned site passed a real-cloud round on 2026-11-21 (real AWS and VEXXHOST, from a laptop): live updates, a split deployment, custom code, scaling and the ten-minute demo (`docs/demo.md`), with both clouds empty after. The dark theme is navy, and the landing media show it.
- Waiting on the client: § What we need from the client. Only stage 7 waits on it.
- Patrii and TierHive wait for a public API (`system-b-adapter-layer.md`). Patrii integration testing waits for it too (`test-suites/README.md`).

## Next

### The order

The stages, their work, tests and "done when" are in `docs/architecture/implementation-plan.md`.

| Stage | Status | Open |
| --- | --- | --- |
| 0. Housekeeping | Done: decisions recorded, merged into the fork's `main` | The pull request to the client's repo, and the message to the client |
| 1. Make A safe (phase 0, below) | Done in code, except item 9 | Item 9. A real AWS deploy of A's runners to confirm item 10. The rest: § Open defects in A |
| 2. A test cloud on the laptop (DevStack) | Done: `npm run devstack` (`backend/devserver/README.md` § OpenStack on this machine (DevStack)) | An Intel Mac and Linux are untested; `create` is macOS-only |
| 3. B's core | Done: `backend/adapter` and `adapter-tables.sql` | Clean-up of orphaned secrets and state. Idempotency keys. `role_template_version` on connections. Single-instance `FailInterrupted` |
| 4. General's OpenStack connector | Done: tested on DevStack and VEXXHOST | Require `https` for auth URLs in the cloud |
| 5. The AWS connector | Done: tested on LocalStack and a real AWS account | The role-template version of A's existing rows. The B runner role in the trust policy must exist before a stack names it (stage 7). When a VM with a mounted volume is replaced, the volume is detached from the running VM and can stay busy until the 5-minute timeout (`stop_instance_before_detaching` would fix it, but also stops a VM when only a volume is removed: a team decision). Removing a VM's network link does not replace the VM (`subnet_id` is optional and computed), so destroying that network then fails |
| 5b. Second review of B | Done | A linked role's role-template version is unknown, because A does not record it (this matters only after the first template change). A bounded role's trust policy cannot be limited by IAM conditions. Real AWS must confirm the run tasks' network ACL |
| 6. The frontend and the whole flow | Done | "New diagram" makes two calls, because A's diagram create ignores `data`. Optional: a last refresh in `useDiagramDeployments`. "Recent runs" lists only runs seen since the page opened, because B returns only the latest run and the last deploy (needs a B route). The editor caps block names at 32 characters while B accepts 64 |
| 6b. No manual steps | Done | A's deployer image and Ansible did not run on LocalStack; they need Clutter hosted |
| 7. The cloud | Written in code. World 2 works from a laptop (`npm run real-aws`, and any real OpenStack project) | Applying it needs the client's accounts. Not yet proven: hosting (world 1), Fargate runs and S3 state |
| 8. More targets | Later | |

Done for A means the 12-step test in `docs/architecture/environments.md` passes in the cloud. Done for the plan means the demo in `docs/architecture/system-b-adapter-layer.md` § What the user sees runs in the cloud.

### How the team uses this file

- **Claim a task** by telling the team which item you are on, and open a draft pull request right away so others see it (root `README.md` § How we work).
- **Edit only your own rows**, plus the "Where it stands" line your merged work changes. Small edits to separate lines merge cleanly; never rewrite the whole file on a feature branch.
- **Whoever merges a pull request** updates the task's Status to done and moves any new defect into "Open defects".

### Phase 0: make the seams safe

Phase 0 is stage 1 (`docs/architecture/implementation-plan.md` § Stage 1). Items 1 to 8 and 10 are done in code. What changed is in `docs/architecture/system-a-as-built.md` § What changed since the capstone. One item is open:

| # | Task | Done when | Owner | Status |
| --- | --- | --- | --- | --- |
| 9 | A lock on customer Terraform state (an S3 backend with locking instead of the synced local file) | the deployer's generated backend block has locking; two parallel runs on one diagram are refused, confirmed in AWS later | free | open |

## Open defects in A

The defects stage 1 fixed in code stay open on the client's `main` until the fork's `main` reaches it. `docs/architecture/system-a-as-built.md` § What changed since the capstone lists them.

Still open, verified at the code:

- **Shared runner role (high).** A's deployer and Ansible runner share one task role. It may assume any customer's Clutter role (`AllowClutterToDeployTerraformRole-*` in any account), and it may read and write the whole Clutter bucket. The external id, which a run knows only for its own organization, is the only guard. So one run can read or overwrite another customer's state and logs. Fix: a per-run role, as B's runner has. A's AWS path retires after B covers it.
- **The Ansible job's SQS message body carries the role ARN and external id (high).** Anyone who can read the queue gets them for every job. No fix is chosen yet. Sending only the job id does not work alone: since stage 1 the runner has no database access, so it cannot look the role up. The fix belongs with the per-run role above (the run's own role, with only that run's values).
- **Customer state has no lock.** It is a local file synced to S3 (phase 0 item 9). Generated code and state for every customer sit in Clutter's own bucket, in the platform's region.
- **Unconfirmed until Clutter is hosted:**
  - The runners' status files: each runner writes a status file to S3 instead of using the database string, and real AWS must confirm those writes (item 10).
  - The SSM document ARN in the customer role, because Ansible runs only once Clutter is hosted (item 7).
- **Customer role upgrades (item 7).**
  - A's EC2 and Lambda roles now sit under `/clutter/` with the boundary. An A deployment made before that would replace those roles on its next apply.
  - An account linked with an older stack must update the stack first.
  - Resources built before the `ManagedBy` default tag lack it, so a version 8 or later role cannot change or destroy them.
  - No user's account is linked in production today, so nothing breaks now.
  - S3 bucket actions stay untagged, because provider 5.x cannot switch on bucket ABAC.
- **Any member can delete an organization.** No invite route exists, so an organization has only its creator. A's delete handler checks membership only, and the website's Danger zone shows Delete to every member. Fix: team roles (an owner role in the database, checked by the delete handler, with the Danger zone shown only to owners).
- A's frontend command type offers `plan`; A's backend accepts only `apply` and `destroy`.
- The frontend's route guard lets any `clutter_token` cookie value through; it never verifies the token. Low impact (page shells only; data needs a real JWT).
- A2's local run signs in with the capstone team's Google client id (`backend/devserver`); the cloud takes the client's own from `google_client_id`.
- **No deploy sends a region.** The account-link page opens AWS in `ca-central-1`, but `runTerraform` sends none, so every deploy runs in the platform's region (`generic.DefaultRegion`). The `us-east-1` default is in the API Gateway JSON model, which API Gateway does not apply.
- `submit-role-arn` stores the ARN with only a format check; B checks the role with STS when it becomes a connection (ADR-019).
- JWT: no `iss` or `aud` claim, and validation is done by hand in `generic.VerifyUserJWT`. An empty secret is refused.
- `logs/live` checks a task's overrides after `DescribeTasks`, not before. A `task_arn` column on `diagram_deployment_logs` would let it check first.
- `role_arn` is in `ansible/list-jobs`'s response code but never written.
- **Canvas drift.**
  - The Lambda `memory_size` choice is ignored: the config panel sends strings and the generator reads numbers, so it is always 128.
  - The code-upload handler is never saved, so Node and Python functions keep `handler = "main"`.
  - The seed's default runtime is `provided.al2`, and the generator's is `provided.al2023`.
- `ansible/submit-job` reports every S3 error as 404 "playbook not found". `POST /terraform-engine` returns 400 "completed with errors" when its templates cannot load, so a platform failure looks like a user error.
- The editor's "save to refresh" needs a change first, because Save is disabled on a clean diagram. Diagrams written through the API load as unsaved, so Preview stays disabled until Save.
- **A's templates.**
  - An API Gateway to Lambda edge never routes traffic: A's API Gateway template renders no method or integration for the target Lambda, and the IAM step only grants the invoke permission.
  - That permission uses a fixed `statement_id`, so AWS rejects a second API on the same function at apply time.
  - A's Lambdas get no basic execution role, so they cannot write logs.
  - The EC2 template pins an x86 image, so an ARM instance type fails.
- `eslint-config-next` stays at 16.0.1: newer versions add React Compiler lint errors in six of A's screens.
- Minor: the Terraform branch of `ansible/run-task` is dead code; `application-tables.sql` fails on a re-run (`CREATE TYPE`).

## Not planned yet

Needed before real users, and not yet designed. Decide each before prod, with the client where it touches money or law.

- **Cost:** who pays for running Clutter (AWS, Supabase, the Fargate run behind every deploy), and whether users pay for Clutter.
- **Legal and privacy:** terms of use, a privacy policy (the website's `/privacy` and `/terms` are drafts for the client's lawyer), and Canadian privacy law (PIPEDA), because Clutter holds access to users' cloud accounts.
- **Teams:** inviting members and roles; today an organization has only its creator.
- **Operations:** monitoring and alerts, backups and restore for Postgres and Terraform state, and who is on call.
- **Leaving:** deleting a user's data, and how a user revokes Clutter's access to their cloud.
- **Abuse and limits:** rate limits, and a cap on deploys, because each one runs a paid worker (B allows at most 3 active runs per organization).
- **Template changes:** a template or provider version change can alter what an already-deployed diagram does on its next deploy.
- **Audit log:** who deployed or destroyed what, and when.

## Later ideas

Not planned or designed yet. Each would be a manifest change in every connector that builds it.

- **Write links:** a link that lets a function write to a bucket or a table, beside today's read links.
- **A load balancer block.**
- **A managed database block.**

## What we need from the client

One message, in one go. Defaults in brackets.

| Need | Why |
| --- | --- |
| An AWS account for Clutter, owned by the client, with team access through user logins [the client owns it] | hosts world 1 in `ca-central-1` (ADR-015) |
| A second AWS account acting as a user, ideally in the same AWS organization [yes] | proves a real cross-account deploy |
| Who pays, a budget cap and a billing alert [$100 CAD for the project] | cloud costs (the team's ENEL 500 Design Roadmap, kept outside this repo, Table 11.1) |
| A Supabase project, or permission to use AWS's own Postgres [Supabase] | the database |
| A Google OAuth client | sign-in |
| A domain and DNS access | the site address, and the Google client's allowed address |
| Agreement that the website runs on Fargate in the same AWS account [yes, ADR-023] | the capstone only built a Docker image |
| Scope sign-off: DevStack as the fallback while Patrii has no public API, and the Design Roadmap's must-have features | the Design Roadmap, section 6.3 |
| Branch protection on the client's `main` (pull requests and the Checks workflow required) [yes] | nothing stops a direct push today |
| A Patrii contact, when the client has one [later] | Patrii's OpenStack endpoints or Terraform provider |

## Open questions

Each has a default we act on until someone answers, and the person or source that settles it. Answered questions are recorded as ADRs.

| # | Question | Until answered, we | Settled by |
| --- | --- | --- | --- |
| 1 | What does General (custom) cover? | a family: one connector per approved Terraform provider, plus existing servers over SSH (ADR-008) | the team and the client |
| 2 | Which providers does General start with after OpenStack? | DigitalOcean | the team and the client |
| 3 | Why TierHive, and does the client know KernelKeepers LTD? | treat TierHive as later: a General connector if it ships a Terraform provider, its own otherwise; its hand-made servers through General's SSH mode once that exists | the client |
| 4 | Is Patrii a place to build, a place to host Clutter, or both? | a place to build | the client (data residency needs would change it) |
| 5 | Where does Terraform state go in B? | in the user's storage when their target has any, otherwise ours, encrypted and locked | the team, in stage 3 |
| 6 | Does the other team's live collaboration need anything from the diagram document or B API? | keep the document's shape stable and tell them before changing it | the other team |
| 7 | Should CI move to `actions/checkout` v6, as the client's `dev` branch does? | yes, the next time CI changes | the team |
| 8 | Are ADR-001 to ADR-028 accepted? Only ADR-012 is. | act on them as written | the team, in one review; the client for scope and hosting (ADR-004, ADR-007, ADR-013, ADR-015) |
| 9 | Should removing a linked AWS account in A also remove the connections made from it? Today a connection keeps its own copy of the role and keeps working until the user deletes the stack in their account. | keep them separate; the Connections page lists each connection with its own Remove | the team |
| 10 | The landing page lists which blocks each cloud builds, a copy of the connectors' manifests. | change the landing copy in the same change as a manifest | the team |
| 11 | In A's editor a node placed at the top edge (the seeded S3 node, at y = 0) sits under the floating Back and Save toolbar, and the seeded diagram opens with Save on and Deploy off until it is saved once. Both come from the capstone's code. | leave A's editor as it is until C replaces it (ADR-021); a fix is a fit-to-view on load | the team |
