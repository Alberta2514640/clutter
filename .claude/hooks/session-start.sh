#!/usr/bin/env bash
# SessionStart: prints where the repo and the work stand, so every session opens oriented. Read-only.
set -uo pipefail
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

echo "Lead session: read .claude/lead.md and .claude/findings/NOW.md whole before the first action."
echo
echo "Branch: $(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
dirty="$(git status --short 2>/dev/null | head -20)"
if [ -n "$dirty" ]; then echo "Uncommitted:"; echo "$dirty"; else echo "Working tree clean."; fi
echo
echo "Recent commits:"
git log --oneline -5 2>/dev/null
if [ -f .claude/findings/NOW.md ]; then
  echo
  echo "NOW.md (head):"
  head -30 .claude/findings/NOW.md
fi
command -v jq >/dev/null 2>&1 || echo "Warning: jq is missing, so the guard hooks block every guarded call until it is installed (brew install jq)."
exit 0
