---
paths:
  - "docs/**/*.md"
  - "**/CLAUDE.md"
  - "**/README.md"
---

# Doc rules (path-scoped)

- **One owner per fact.** `docs/README.md` maps each kind of fact to its one file. Change a fact at its owner, in the same change as the code that moved it; never paste it into a second doc.
- **Code beats doc.** When a doc and the code disagree, the code is true. Fix the doc, do not trust it.
- **No file-by-file inventories or `file.go:123` pointers in docs.** They stop resolving silently. Name the owner (a package, a function, a folder) instead. Pointers belong in replies and reviews, not in docs.
- **Present tense.** A doc says what is true now. What changed and why lives in git and `docs/decisions.md`.
- **Research is dated.** A file in `docs/research/` starts with its date and its sources, and is superseded by a newer dated file, not edited into a different claim.
- **Plain everyday English.** Short complete sentences, no em dashes in anything a person outside the team reads.
- `README.md` at the root is the final course report (2026-06-08). Treat it as a historical record of the design, not as the current architecture; `docs/architecture/system.md` owns that.
