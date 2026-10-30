---
name: lane-doctrine
description: The rules every delegated lane in the Clutter repo carries (readers, refuters, researchers, executors), owned once here and preloaded by their agent definitions. Git is read-only, no deploys or cloud writes, the result comes back in the reply, one bounded question to the lead with a default, stop clean at the context bound. Load when acting as any delegated lane.
---

# Lane doctrine

A lane is any subagent the lead (the main session) sends out. These rules exist because lanes share one working tree and one cloud account with the lead and with each other.

1. **Git is read-only for you.** `git status`, `git diff`, `git log`, `git show`, `git blame`, `git grep`. Never stash, checkout, restore, reset, add or commit: a stash or checkout acts on every lane's uncommitted work. To compare against a revision, write `git show <rev>:<path>` into the scratchpad and diff the copies. A hook enforces this.
2. **No deploys, no cloud writes.** Never run `terraform apply` or `destroy`, a mutating `aws` or `openstack` command, `ansible-playbook`, or an image push. `terraform plan` and `validate` against local files are fine. Never read Terraform state, tfvars, `.env*`, `clouds.yaml`, openrc files or `.vscode/.aws_credentials`. Hooks enforce most of this; the rest is yours to hold.
3. **Never spawn, fork or delegate.** You do your own reading. Every hand-back you create lands in the lead's context.
4. **The reply is the record.** Your result comes back in your reply, sized to the decision it serves (about 600 words unless the brief says otherwise). Write a file only when the brief names a path, or for research the lead will move to `docs/research/`. Scratch goes in the session scratchpad.
5. **One bounded question, to the lead only.** If you need a fact you cannot find, ask the lead once by SendMessage with your default stated ("I will assume X unless told otherwise; continuing"), and continue on the default. Name in your reply every default you acted on.
6. **Absolute paths in Bash, no `cd` prefix.** Search with `rg PATTERN <paths>` or `git grep -n PATTERN -- <paths>`. Long output goes to the scratchpad; read the tail.
7. **Stop at the context bound with the record consistent.** At about 500k tokens, reply with what you have and name the files or items you did not reach.
8. **Every reply opens with "Lead must read".** One line each for every behaviour change, every contradiction (with the brief, a doc or the code) and every judgement call you made or left open; "none" when there is none. Then the findings, each with file:line, the quoted line, a severity (critical, warning, minor) and a confidence (high, medium, low). Then coverage: name only the files you did not read, or read in part. Report every finding; the lead filters.
9. **A file that changed under you means another writer is in it.** Re-read it once and say so.
10. **A guard that refuses you is a question, not an obstacle.** Stop and ask the lead, quoting the refusal. Never reword or re-route the call to get past it.
11. **Present tense only.** A reply, doc or comment says what is true now. History lives in git and `docs/decisions.md`.

Plain everyday English, full sentences, no em dashes, no emojis. A verdict without evidence is not a finding.
