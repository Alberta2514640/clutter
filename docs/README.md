# Docs map

Each kind of fact has one owner. Change a fact at its owner, in the same change as the code that moved it. When a doc and the code disagree, the code is true and the doc is fixed.

| Fact | Owner |
|---|---|
| How agents behave in this repo | `CLAUDE.md` (root and per folder) |
| Code rules | `.claude/skills/engineering-conventions/SKILL.md` |
| The architecture as it stands | `docs/architecture/system.md` |
| The provider adapter design and its phases | `docs/architecture/provider-adapter.md` |
| What was decided, and why | `docs/decisions.md` |
| Outside research (cloud providers, prior art, law) | `docs/research/<date>-<topic>.md` |
| Where the current work stands, open defects | `.claude/findings/NOW.md` |
| A resource's fields and defaults | the generator code today (and, drifting from it, the seed SQL and the config panel); the provider's resource manifest once the adapter lands |
| The original design and course deliverable | root `README.md` (final report, 2026-06-08; history, not current) |
