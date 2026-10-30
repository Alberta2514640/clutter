# Docs map

Each kind of fact has one owner. Change a fact at its owner, in the same change as the code that moved it. When a doc and the code disagree, the code is true and the doc is fixed.

| Fact | Owner |
| --- | --- |
| How agents behave in this repo | `CLAUDE.md` (root and per folder) |
| Code rules | `.claude/skills/engineering-conventions/SKILL.md` |
| What exists today (System A), what was delivered, and how to run and check it | `docs/architecture/system.md` |
| What we are building (System B, the adapter layer), its names and its phases | `docs/architecture/provider-adapter.md` |
| The team's own architecture figures (Options 1 to 3, B1 and B2) | `research_notes/Re-Clutter Architecture/` (source material; the design doc owns the decisions) |
| The second research run on Patrii and Clutter (2026-10-26) | `reports/` and `research_notes/PatriCloud Clutter adapter layer/` (source material, folded into the docs above) |
| What was decided, and why | `docs/decisions.md` |
| Outside research (cloud providers, prior art, law) | `docs/research/<date>-<topic>.md` |
| Where the current work stands, open defects | `.claude/findings/NOW.md` |
| A resource's fields and defaults | the generator code today (and, drifting from it, the seed SQL and the config panel); the provider's resource manifest once the adapter lands |
| Onboarding: what Clutter is, setup, where to start | root `README.md` |
| The original design and course deliverable | `docs/history/2026-06-08-final-report.pdf` and `.md` (same report; history, not current; its figures were never committed) |
