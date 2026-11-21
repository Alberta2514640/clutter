# Docs map

Each kind of fact has one owner. Change a fact at its owner, in the same change as the code that moved it. When a doc and the code disagree, the code is true and the doc is fixed.

| Fact | Owner |
| --- | --- |
| How agents behave in this repo | `CLAUDE.md` (root and per folder); `AGENTS.md` points other AI tools at them |
| What each part of the Claude Code setup does | `.claude/README.md` |
| Code rules | `.claude/skills/engineering-conventions/SKILL.md` |
| What exists today (System A): how it runs, what stands in for AWS locally, what changed since the capstone, what it can and cannot do | `docs/architecture/system-a-as-built.md` |
| Onboarding: what Clutter is, the ways to run it, the setup sequence for a new person, what not to run | root `README.md` |
| The git workflow: branches, where a pull request goes, taking a task | root `README.md` § How we work |
| The ten-minute demo script | `docs/demo.md` |
| Running A and B locally (A2), with LocalStack for AWS and DevStack for OpenStack; the smoke test's results; what works locally and what only works in AWS | `backend/devserver/README.md` |
| Backend layout, checks and their results, A's Go and shell lint, adding a Lambda | `backend/README.md` |
| B's service (the adapter layer): its packages, `/v1` routes, settings, function code, how it runs locally and in the cloud, and its checks | `backend/adapter/README.md` |
| Which cloud logos and service icons the website may show, their sources and terms | `frontend/clutter/public/clouds/README.md` |
| Frontend layout, run variants, checks and their results, browser results | `frontend/clutter/README.md` |
| The generator's internals | `backend/api/terraform-engine/create/README.md` |
| What differs between local and the cloud, what works where, and how to set up the cloud | `docs/architecture/environments.md` |
| The order B is built in, each stage's tests and "done when", and what can be tested without cloud accounts | `docs/architecture/implementation-plan.md` |
| The worlds, delivery models and environments in one picture | `docs/architecture/figures/clutter-worlds.png` (shown in `system-b-adapter-layer.md`) |
| C after A's AWS path retires, in one picture | `docs/architecture/figures/clutter-c.png` (shown in `implementation-plan.md`) |
| The building-block editor's screenshot | `docs/architecture/figures/editor.png` (shown in the root `README.md`) |
| What we are building (System B, the adapter layer), its names and its design | `docs/architecture/system-b-adapter-layer.md` |
| The team's own A/B plan: Options 1 to 3, B0, B1 and B2, its figures and its explanation | `docs/architecture/system-b-team-plan/` (source material; `system-b-adapter-layer.md` owns the details) |
| The second research run on Patrii and Clutter | `docs/research/patricloud/2026-10-26-second-research-run/` (source material, folded into the docs above) |
| What was decided, and why | `docs/decisions.md` |
| Outside research (cloud providers, prior art, law) | `docs/research/<target>/<date>-<topic>.md` (`patricloud/`, `tierhive/`, `openstack/`, `aws/`, which holds the LocalStack research, and `website/`, which holds the UX research behind ADR-028) |
| Where the current work stands, open defects | `.claude/findings/NOW.md` |
| Dated test reports, with screenshots, for people who will not run the tests | `docs/reports/<date>-*.pdf` |
| The team's test plan: one suite per test type, where each lives and how to run it | `test-suites/README.md` |
| A resource's fields and defaults | in A, the generator code (and, drifting from it, the seed SQL and the config panel); in B, each connector's manifest |
| The previous capstone team's design and course deliverable | `docs/old-capstone/`: `final-report.pdf` and `.md` (the same report; history, not current) and `backend-tutorial.md` (the capstone's backend guide) |
