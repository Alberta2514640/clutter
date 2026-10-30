# Start here (any AI tool)

This file points every coding agent (Claude Code, Cursor, Codex, or the next one) at the same canon. It is a pointer, not a second copy.

People new to the repo start at the root `README.md` instead.

1. **How to behave here:** root `CLAUDE.md`. Folder `CLAUDE.md` files add local rules where you work.
2. **Code rules:** `.claude/skills/engineering-conventions/SKILL.md`. Read it before writing or changing code.
3. **What is true:** `docs/README.md` maps each fact to its one owner; `docs/architecture/system-a-as-built.md` is the architecture as it stands.
4. **Hard lines, whatever your tool:** never read `.env*`, Terraform state, tfvars, `clouds.yaml`, openrc files or `.vscode/.aws_credentials`; never run `terraform apply`, a mutating cloud CLI command or an image push without the person's approval. Claude Code enforces these with hooks; other tools do not, so hold them yourself.
5. **Skills:** Claude Code loads `.claude/skills` on its own. Other tools: when a task matches a skill's description, open `.claude/skills/<name>/SKILL.md`.
