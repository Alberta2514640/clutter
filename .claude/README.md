# .claude: the AI agent setup

This folder sets up Claude Code for this repo. You do not need it to work on Clutter, and nothing in it ships. Most folder names here are fixed by Claude Code; the two that are ours are `lead.md` and `findings/`.

| Path | What it is | Who reads it |
| --- | --- | --- |
| `findings/NOW.md` | **The status board**: the current goal, the next tasks in order, every open defect in A, and the open questions. Read it before picking up work. | everyone |
| `findings/README.md` | what else may live in `findings/` (scratch notes, ignored by git) | everyone |
| `settings.json` | Claude Code settings: which commands are allowed or blocked, and which hooks run when | Claude Code |
| `hooks/` | scripts Claude Code runs around its actions (table below). They need `jq` installed; without it the guards are off. | Claude Code |
| `agents/` | the four helper agents: `reader` (reads and reports, never edits), `researcher` (outside facts with sources), `executor` (builds a planned change), `red-pill-reviewer` (tries to break finished work) | Claude Code |
| `skills/` | step-by-step procedures: `engineering-conventions` (code rules), `add-cloud-resource` (every place a canvas resource is defined), `ground-before-building` (check current docs first), `lane-doctrine` (rules for helper agents). Worth reading yourself before writing code. | Claude Code, and people |
| `rules/` | reminders that apply to one kind of file: `docs.md` (Markdown docs), `terraform.md` (Terraform and templates) | Claude Code, and people |
| `lead.md` | how the main Claude session splits and checks work | Claude Code |

## The hooks

| Hook | When it runs | What it does |
| --- | --- | --- |
| `session-start.sh` | a session starts | prints the branch, uncommitted files, recent commits and the top of `NOW.md` |
| `guard-secrets.sh` | before a file read or a shell command | blocks reading `.env*`, Terraform state, tfvars, cloud credential files and private keys |
| `guard-cloud-writes.sh` | before a shell command | asks you first before `terraform apply` or `destroy`, a mutating `aws` or `openstack` command, `ansible-playbook`, or an image push |
| `guard-git-writes.sh` | before a shell command from a helper agent | keeps git read-only for helper agents |
| `format-on-write.sh` | after a file is written | runs `gofmt` or `terraform fmt` on new files and files already formatted at the last commit; leaves A's unformatted and CRLF files alone |
| `lib.sh` | (shared) | helpers the hooks use |
| `test-hooks.sh` | by hand, after changing a hook | tests every hook; must report 0 failed |

The rules every agent follows are in the root `CLAUDE.md` (and the folder `CLAUDE.md` files); `AGENTS.md` points other AI tools at the same rules.
