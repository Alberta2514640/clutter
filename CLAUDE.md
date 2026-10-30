# CLAUDE.md

The frames to hold before opening a file, the boundaries not to cross, and where everything else lives. Folder `CLAUDE.md` files load where you work; `docs/README.md` maps each fact to its one owner. How the lead (the main session) runs the work is `.claude/lead.md`. Nothing here restates what `ls`, `grep` or a type signature already answers.

## What Clutter is

A drag-and-drop canvas that turns an architecture diagram into Terraform (and Ansible), then runs it inside the customer's own cloud account. It began as a University of Calgary capstone (team 500-MM-3, client Alberta 2514640 Inc); the root `README.md` is that final report, dated 2026-06-08, and is history, not the current architecture (`docs/architecture/system.md` owns that).

The work ahead is a **central standard layer** with two faces: a versioned inbound API that people and other products call to use Clutter, and outbound provider adapters so Clutter can deploy to more than AWS, starting with Patrii Cloud (a Canadian OpenStack cloud). The design is `docs/architecture/provider-adapter.md`; what we know about Patrii is `docs/research/2026-10-26-patrii-cloud.md`.

## What you will get wrong about this repo

- **There are two Terraforms.** `backend/infra` deploys Clutter itself into Clutter's AWS account. The `*.tf.tmpl` files are customer templates the generator renders into code that runs in the customer's account. A change to one never stands in for the other (`.claude/rules/terraform.md`).
- **Templates do not ship with the Lambda.** They are uploaded to S3 by the platform Terraform (one `aws_s3_object` per file) and fetched at run time. A new template file that is not added there does not exist in production.
- **A resource is defined in five places that must agree:** the generator's type switch and generator file (Go), the template, the `supported_resources` seed (SQL), the canvas palette and config panel (frontend), and the S3 upload. They already drift (runtimes, regions, defaults). The adapter work exists to collapse these to one owner per provider; until then, `add-cloud-resource` lists them all.
- **"AWS" is not only in the templates.** It is in the core: the `provider "aws"` block the writer emits, IAM policy built from edges, the region list, the account-link flow (CloudFormation and role ARN), the runner scripts (STS and SSM), the DB columns, and the frontend's node component. Assume any file may hold an AWS assumption until you have read it.
- **Nothing in this repo triggers generation.** `POST /terraform-engine` expects a Supabase database-webhook body, and that webhook is configured in Supabase, not here. Neither the frontend nor the diagram handlers call it.

## Boundaries, and what enforces them

- **Tenant isolation is the product's first promise.** Every handler that takes an org, project, diagram, run or account id checks that the caller is a member of the org and that the child belongs to that org. Membership alone is not enough. A handler without that check is a defect, whatever else it does.
- **User input never reaches generated code unescaped.** A value from the canvas that lands in a template, an IAM policy, an Ansible playbook or a shell command is validated against an allow-list or escaped for that target. Generated code runs with the customer's credentials and the runner holds platform secrets, so an injection here is a cross-tenant breach.
- **Customer credentials and Terraform state never reach a log, a response, a session transcript or a doc.** Hooks refuse reading `.env*`, state, tfvars, `clouds.yaml`, openrc files and `.vscode/.aws_credentials`.
- **No session deploys.** `terraform apply` or `destroy`, a mutating `aws` or `openstack` command, `ansible-playbook` or an image push goes to the person for approval (the `guard-cloud-writes` hook asks). Lanes never deploy.
- **The provider-neutral core imports no provider.** As the adapter layer lands, AWS and Patrii code live behind the provider interface; core code that branches on a provider name is a defect, not a shortcut (`docs/architecture/provider-adapter.md`).

## Known open defects

The 2026-10-26 audit found security defects that are still open: an unauthenticated generation endpoint, template injection, an over-broad customer role, and ownership checks that stop at membership. They are listed with their fixes in `.claude/findings/NOW.md`. Do not build on top of a path they touch without fixing or flagging it.

## Commands

Run from the repo root with absolute paths; no `cd dir && cmd`.

| Question | Command | Notes |
|---|---|---|
| Does the Go compile? | `go -C backend/api build ./...` and `go -C backend/api vet ./...` | Both green as of 2026-10-26. |
| Do the Go tests pass? | `go -C backend/api test ./...` | Two packages fail today: an integration test that needs live S3, and a nil-connection panic. Read the per-package lines; a red run is not a reason to skip the packages you touched. |
| Does the frontend type-check and lint? | `npm --prefix frontend/clutter run lint`, `npx --prefix frontend/clutter tsc --noEmit -p frontend/clutter` | Needs `npm ci` in `frontend/clutter` first. npm is the real package manager (the Dockerfile runs `npm ci`); `pnpm-lock.yaml` is stale. No frontend tests exist. |
| Is the platform Terraform valid? | `terraform -chdir=backend/infra fmt -check -recursive`, then `validate` after `init -backend=false` | Never `plan` against the real backend from a session. |
| Do the hooks still work? | `.claude/hooks/test-hooks.sh` | Run after changing any hook. |

CI (`.github/workflows/docker-image.yml`) only builds and pushes the frontend image. Nothing else is checked on push, so the commands above are the only gate.

## When you are uncertain

- A search that returns nothing is the claim most worth double-checking. Say "I have not checked" rather than infer absence.
- Ambiguity about architecture, tenancy, credentials, the schema or user-visible copy: ask, with a default, and keep working on the default.
- Patrii Cloud is in beta and changes monthly. Re-check any Patrii fact older than a month against patrii.cloud before building on it (`ground-before-building`).

## Habits that matter here

- **Grep to locate, read to understand.** A grep says a symbol occurs; only the surrounding lines say what is true.
- **Follow a change through every layer it rides:** canvas node, API schema, Go handler, generator, template, S3 upload, runner. A field added in three of five places is the drift this repo already has.
- **Fix the class at its owner.** A missing ownership check in one handler is almost always missing in its siblings; sweep them all.
- **Every narrowing change drops something real.** Before adding a filter, guard or allow-list, name the legitimate input it now refuses.
- **Delete what a change makes unused** in the same change. No `_v2` copies beside the original.

## Skills

Skills route by their own descriptions. The ones a description alone may not catch:

| Doing this | Use |
|---|---|
| Writing or changing any code | `engineering-conventions` |
| Starting a feature, or touching a cloud API, Terraform provider or library | `ground-before-building` |
| Adding or changing a canvas resource (any provider) | `add-cloud-resource` |
| Running as a delegated lane | `lane-doctrine` (preloaded by the agents) |

## Which surface a thing belongs on

| Surface | Holds | Write here when |
|---|---|---|
| this file | how to think and behave here | a behaviour must change |
| `.claude/hooks/` | what must never go wrong, enforced | the model must not be able to get it wrong |
| `.claude/skills/` | how to do a repeatable job | a procedure changes |
| `.claude/rules/` | path-scoped reminders | a rule applies to one kind of file |
| `docs/` | what is true | a fact changes |
| `docs/decisions.md` | what was decided and why | a decision is made |
| `docs/research/` | dated outside research | a source was read |
| `.claude/findings/NOW.md` | where the work stands | work lands, the plan changes, or the session ends |
| a code comment or folder `CLAUDE.md` | the trap a line or folder must not fall into | an invariant would surprise the next editor |

The test for a line in this file: would removing it cause a model to repeat a mistake? If not, it belongs elsewhere or nowhere.

## Repo-wide rules

- **No em dashes** in user-facing copy, docs a person outside the team reads, or chat replies. Code comments may follow the file's style.
- **Plain everyday English** wherever a person reads it.
- **Never commit, push or open a PR unless the person asks.** Work happens on a branch, never straight on `main`. Commit messages describe what changed and why; no tool attribution.
