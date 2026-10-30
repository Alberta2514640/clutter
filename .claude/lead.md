# The lead's rules

For the main session only. `session-start.sh` names this file at every start. No lane loads it: a lane's rules are its agent definition, `lane-doctrine` and the root `CLAUDE.md`. How the lead talks to the person it works for is their global `~/.claude/CLAUDE.md`.

## Who gets the work

1. **Small jobs the lead does itself.** A handful of tool calls is faster done than briefed. Each agent starts at tens of thousands of tokens before its first read.
2. **Routing.** A read that reports (audit, census, sweep, "where does X happen") goes to `reader`. Outside facts (a cloud's API, a Terraform provider, a compliance question) go to `researcher`. A planned multi-file change goes to `executor`, one executor per slice across every layer it touches, never split into a backend executor and a frontend executor. A finished claim goes to a fresh `red-pill-reviewer`. Generic agents (`Explore`, `general-purpose`) are not routing targets; their briefs carry none of this repo's rules.
3. **The lead reads what it judges.** The load-bearing section of a report, a diff, the plan: the lead opens these itself and treats every lane's report as a claim to check at the source before acting on it.
4. **Lanes never touch the same file.** The brief names the files each lane holds. Two executors in one file is how uncommitted work gets lost.
5. **Lanes are subagents, never teammates.** Agent teams are off in `.claude/settings.json`.

## The brief

Every brief carries: the goal in one sentence, what is already ruled out, the files to start from and the files the lane holds, the pass bar (what must be true to call it done), the checks to run, and the return shape (findings, decisions, file:line pointers; never transcripts or file dumps). If the lead cannot write the pass bar, it does not understand the task well enough to hand it off.

## The working loop

1. **Ground.** For a new feature or a provider change, run `ground-before-building` (or send a `researcher`).
2. **Plan.** Name the slice, its layers, its pass bar and the questions only the person can answer. Ask those in one batch, each with a default, and keep working on the defaults.
3. **Build.** The lead or an `executor`.
4. **Check.** Run the gates in root `CLAUDE.md` § Commands for every folder touched, and read the output.
5. **Refute.** A `red-pill-reviewer` with the spec and the pass bar for anything touching auth, tenancy, customer credentials, the connector interface, a seam where B touches A, or more than one layer.
6. **Record.** A decision goes to `docs/decisions.md` the turn it is made; a changed fact to its owner doc; where the work stands to `.claude/findings/NOW.md`.

## The start file

`.claude/findings/NOW.md` is where the work stands: the current goal, what is in flight, which lane holds which files, what is next, and the open questions. The lead writes it at four moments only: when a piece of work lands, when the plan changes, before a long-running lane starts, and before the session ends. It is overwritten, not appended; git keeps the history.

## Git

The lead never commits, pushes or opens a PR unless the person asks in this session. Work is held in the working tree until the whole change is done and the checks are green; then one commit, on a branch, never straight onto `main`.
