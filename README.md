# Clutter

Clutter is a drag-and-drop canvas for cloud architecture. You draw a diagram (a Lambda, an S3 bucket, an EC2 server, with lines between them), and Clutter turns it into Terraform and Ansible and runs it inside your own cloud account.

It began as a University of Calgary capstone (team 500-MM-3, 2025 to 2026) for Alberta 2514640 Inc. That version works with AWS only. We are now extending it so a diagram can deploy to other targets, such as PatriCloud and TierHive, through an adapter layer.

This page gets you from a fresh clone to a working setup, and shows where everything lives. Read it once, top to bottom.

## Where the project stands

| | |
| --- | --- |
| **System A** (Clutter as built, AWS only) | Complete and tested by the capstone team. Its AWS deployment has been torn down. It runs locally through `backend/devserver`, except the parts that need AWS (generate, deploy, logs, Ansible). |
| **System B** (the adapter layer) | Designed, not built. Phase 0 (security fixes A and B share) comes first. |
| **Local checks** | Go build, frontend type-check, lint and build, Terraform validate and the database schema all pass. Two Go test packages fail (known). |
| **Open defects** | Several security defects in A, some critical. Listed in `.claude/findings/NOW.md`. Read that before you build on any A code path. |

## The two systems

**A: Clutter today.** A Next.js canvas, about 31 Go Lambdas behind AWS API Gateway, Postgres on Supabase, S3, and two Fargate runners: one runs Terraform in the customer's AWS account, one runs Ansible on the customer's EC2 servers over SSM. Full description: `docs/architecture/system.md`.

**B: the adapter layer.** New code beside A, not a rewrite of A. B has its own API, a connector interface, and one connector per target: General (custom) first, then PatriCloud and TierHive. Full design: `docs/architecture/provider-adapter.md`. The team's original figures are in `research_notes/Re-Clutter Architecture/`.

The names everyone uses:

| Name | Means |
| --- | --- |
| A | Clutter as built (AWS). We change it only at four named seams. |
| B | the adapter layer we are building |
| B1 | B as a service anyone can run, with entity connectors (built first) |
| B2 | B run inside a company, with instance connectors (later) |
| connector | the code for one target type (General, PatriCloud, TierHive) |
| instance | one company's configured setup of a connector |
| seam | one of the four places B touches A: the target choice screen, shared sign-in, the shared database, and security fixes |

## What is in the repo

```text
backend/
  api/              Go Lambdas, one folder per route; generic/ is shared code
    terraform-engine/create/   the generator: diagram JSON to Terraform
  infra/            Terraform that deploys Clutter itself into Clutter's AWS account
    modules/s3/uploads/        the customer Terraform templates (*.tf.tmpl)
    modules/fargate/ecr-scripts/   the two runner images
  storage-sql/      the Postgres schema and the resource seed
  devserver/        dev-only: runs A's real Lambdas locally behind an API Gateway stand-in
  tfstate-infra/    bootstraps the remote state bucket for infra/
frontend/
  clutter/          the Next.js app (canvas, dashboard, settings)
  Dockerfile.frontend
docs/
  architecture/     system.md (A as built), provider-adapter.md (B, planned)
  decisions.md      every decision and why (ADR-001 onward)
  research/         dated outside research (Patrii, TierHive)
  history/          the capstone final report, PDF and markdown
research_notes/, reports/   source material behind the design
.claude/            the Claude Code setup: rules, agents, skills, hooks
CLAUDE.md, AGENTS.md        instructions for AI coding agents (worth reading yourself)
```

About 20,000 lines of code: 97 Go files, 119 TypeScript files, 40 Terraform files.

There are two kinds of Terraform here. `backend/infra` deploys Clutter itself. The `*.tf.tmpl` files are templates the generator fills in for a customer. They never stand in for each other.

## Set up

### 1. Install

| Tool | Version | Needed for |
| --- | --- | --- |
| Git and GitHub CLI (`gh`) | any recent | everything |
| Go | 1.25 or newer | the backend |
| Node.js with npm | 22 or newer | the frontend (use npm; `pnpm-lock.yaml` is stale) |
| Terraform | 1.x | checking the infrastructure code |
| Docker (running) | any recent | the local database |
| curl, openssl, python3 | macOS has them | the local run scripts |
| AWS CLI | v2 | only for deploying, which nobody does from a laptop yet |
| Claude Code | optional | the AI setup in `.claude/` |

### 2. Clone

Our working repo is a fork. The client's original is `upstream`; never push to it.

```bash
git clone https://github.com/FayzanT/re-clutter
cd re-clutter
git remote add upstream https://github.com/Alberta2514640/clutter
npm --prefix frontend/clutter ci
```

### 3. Check that everything builds

Run from the repo root.

```bash
go -C backend/api build ./...
go -C backend/api vet ./...
go -C backend/api test ./...          # 2 packages fail today; see below
npx --prefix frontend/clutter tsc --noEmit -p frontend/clutter
npm --prefix frontend/clutter run lint
NEXT_PUBLIC_API_ENDPOINT=http://localhost:9999 npm --prefix frontend/clutter run build
terraform -chdir=backend/infra fmt -check -recursive    # 7 files fail today
go -C backend/devserver test ./...     # the local run tool
```

The two failing test packages are known: `terraform-engine/create` needs live S3 (it waits about 70 seconds, then fails), and `tests/generic` panics on a nil database connection. Every other package must pass. The expected result of each check is kept up to date in `docs/architecture/system.md` § Running it.

### 4. Run the app locally

You need two terminals. The first run takes a minute or two while it builds the 33 Lambdas.

Terminal 1, the database and backend:

```bash
backend/devserver/run.sh                     # Postgres, then A's real Lambdas behind a local API Gateway
```

Terminal 2, the frontend (it keeps running):

```bash
NEXT_PUBLIC_API_ENDPOINT=http://localhost:8787 npm --prefix frontend/clutter run dev -- --hostname 127.0.0.1 --port 3100
```

Then:

1. Open <http://localhost:3000/dev/login>. It signs you in as a local dev user. (Port 3000 is the devserver's front door; it passes pages through to the frontend on 3100.)
2. The first time, the database is empty. Create an organization at <http://localhost:3000/onboarding/create-org>, then a project from the dashboard.
3. The diagram editor stays locked until the organization has a linked AWS account. Link a fake one (it only writes to the local database):

   ```bash
   backend/devserver/link-fake-account.sh
   ```

4. Open a diagram and use the canvas. Deploy will fail at its AWS call; that is expected.

Other commands:

```bash
backend/devserver/smoke.sh                   # calls all 31 routes and prints what passed
backend/devserver/stop.sh                    # stops everything; add --wipe to delete the local database
```

Organizations, projects, diagrams, the canvas and the resource palette work locally. Generating Terraform, deploying, logs and Ansible only work in AWS. The full table is in `backend/devserver/README.md`.

### 5. Look at the UI without the backend

Stop the devserver first (`backend/devserver/stop.sh`); both use port 3000.

```bash
NEXT_PUBLIC_API_ENDPOINT=http://localhost:9999 npm --prefix frontend/clutter run dev
```

Open <http://localhost:3000>. The home page and `/login` render. You cannot sign in this way; use step 4.

### 6. Load the database by hand

`run.sh` does this for you. To do it by hand:

```bash
docker run -d --rm --name clutter-pg -e POSTGRES_PASSWORD=local -e POSTGRES_DB=clutter postgres:17-alpine
until docker exec clutter-pg pg_isready -U postgres -d clutter; do sleep 1; done
docker exec clutter-pg psql -U postgres -d clutter -c "create role anon; create role authenticated; create role service_role;"
docker exec -i clutter-pg psql -U postgres -d clutter < backend/storage-sql/application-tables.sql
docker exec -i clutter-pg psql -U postgres -d clutter < backend/storage-sql/seed-resources.sql
docker exec clutter-pg psql -U postgres -d clutter -c "\dt"
docker stop clutter-pg
```

You should see 11 tables. The roles exist because the schema was written for Supabase.

## What only works in AWS

The Go code was built to run as AWS Lambdas; `backend/devserver` runs it locally, but anything that calls S3, ECS or SQS stops there. The capstone deployment is gone, and its AWS account is not ours. Running Clutter fully, including deploys, needs our own AWS account, a Postgres or Supabase database, a Google sign-in client, and a set of secrets. No extra tools are needed. The nine steps, and what must be fixed first, are in `docs/architecture/system.md` § Running it.

Do not run:

- `frontend/clutter/init.bash`. It is an old scaffold script that overwrites real source files.
- `terraform plan` or `apply` in `backend/infra`. Its state backend is the capstone team's AWS account.

## Your first day

Read in this order:

1. This page.
2. `docs/architecture/system.md`: what A does, what the capstone delivered, and where the final report is wrong about the code.
3. `docs/architecture/provider-adapter.md`: what we are building and in what order.
4. `.claude/findings/NOW.md`: what is in progress, the open defects, the open questions.
5. `docs/decisions.md`: why things are the way they are.
6. `CLAUDE.md`: the traps people (and AI agents) fall into in this repo.

Then pick a phase 0 item from `NOW.md`.

## How we work

- **Branches.** Branch off `main`, never commit to it directly, and open a pull request against our fork (`origin`). Never push to `upstream`.
- **Commits.** Say what changed and why. One commit per finished change, after the checks pass.
- **Changing A.** A changes only at the four seams. Any other change to A is a decision: write it in `docs/decisions.md` first.
- **Security rules.** Every handler that takes an id checks that the caller belongs to the organization and that the thing belongs to that organization. No value from the canvas reaches generated code without validation or escaping. Credentials and Terraform state never go in logs, responses or docs.
- **Docs.** Each fact has one owner doc (`docs/README.md` has the map). When code changes a fact, update its owner in the same pull request. When a doc and the code disagree, the code wins and the doc gets fixed.
- **Adding a cloud resource.** Follow `.claude/skills/add-cloud-resource/SKILL.md`, even without Claude Code. It lists every place a resource is defined.

### Claude Code

The `.claude/` folder sets up Claude Code for this repo. You do not need it, but if you use it:

- `CLAUDE.md` is read at the start of every session; folder `CLAUDE.md` files load when Claude works in that folder.
- Agents: `reader` (reads and reports), `researcher` (outside facts with sources), `executor` (builds a planned change), `red-pill-reviewer` (tries to break finished work).
- Skills: `engineering-conventions`, `add-cloud-resource`, `ground-before-building`, `lane-doctrine`.
- Hooks block reading secrets (`.env`, Terraform state, tfvars, cloud credential files), ask before any deploy or cloud write, keep git read-only for sub-agents, and format Go and Terraform on save. After changing a hook, run `.claude/hooks/test-hooks.sh`.

## Where to find things

| Question | Look in |
| --- | --- |
| How does A work today? | `docs/architecture/system.md` |
| What are we building next, and in what order? | `docs/architecture/provider-adapter.md` |
| What is in progress, broken, or undecided? | `.claude/findings/NOW.md` |
| Why was something decided? | `docs/decisions.md` |
| What do we know about PatriCloud or TierHive? | `docs/research/` |
| Which doc owns which fact? | `docs/README.md` |
| What did the capstone team deliver? | `docs/history/2026-06-08-final-report.pdf` |

## History and license

The capstone team: Santiago Garza Fuentes (project manager), Hamza Amar, Maitra Patel, Nimna Wiedasa, Syril Jacob and Akshpreet Singh, advised by Lorenzo De Carli, for Devin Atkin at Alberta 2514640 Inc. Their final report is in `docs/history/`; its figures were never committed.

The capstone code is released under the MIT License, copyright 2026 Alberta 2514640 Inc. The full text is at the end of the final report.
