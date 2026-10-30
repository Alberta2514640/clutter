#!/usr/bin/env bash
# PreToolUse(Bash): in a subagent, git is read-only.
# Lanes share one working tree. A lane's stash, checkout, reset or commit acts on every lane's
# uncommitted work, not only its own files, so git writes belong to the lead (the main session).
set -uo pipefail
. "$(dirname "$0")/lib.sh"

[ "$(hook_field .tool_name)" = "Bash" ] || exit 0
[ -n "$(hook_field .agent_id)" ] || exit 0
cmd="$(hook_field .tool_input.command)"

parts="$(split_commands "$cmd")"
while IFS= read -r part; do
  set -- $part
  [ "${1:-}" = "git" ] || continue
  shift
  # Skip global options such as -C <dir> or -c key=value to reach the verb.
  while [ $# -gt 0 ]; do
    case "$1" in -C|-c) shift 2 ;; -*) shift ;; *) break ;; esac
  done
  verb="${1:-}"
  case "$verb" in
    ""|status|diff|log|show|blame|grep|ls-files|rev-parse|cat-file|merge-base|rev-list|ls-tree|describe|show-ref|for-each-ref|shortlog|check-ignore|ls-remote|whatchanged|name-rev) ;;
    stash)
      case "${2:-}" in list|show) ;; *) deny "git stash in a lane captures every lane's uncommitted work. Git writes are the lead's." ;; esac ;;
    branch|remote)
      [ $# -le 1 ] || case "${2:-}" in -a|-r|-v|-vv|--list|--show-current|show|get-url) ;; *) deny "Branch and remote changes are the lead's." ;; esac ;;
    *) deny "In a lane, git is read-only (status, diff, log, show, blame, grep). '$verb' acts on the shared tree; the lead does git writes. Use 'git show <rev>:<path>' into the scratchpad to compare." ;;
  esac
done <<EOF
$parts
EOF
exit 0
