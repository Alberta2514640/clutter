#!/usr/bin/env bash
# PreToolUse(Read|Grep|Bash): keeps secret files out of the transcript.
# Clutter handles cloud credentials for itself and for customers (AWS keys, Terraform state, OpenStack
# clouds.yaml and openrc files, JWT keys). Anything printed into a session is kept in the transcript.
# Refused: reading, searching or printing a secret file. Allowed: listing, testing, moving or removing one,
# and the committed examples (*.example, *.sample, *.template).
# Not caught: a path built at run time, or a secret behind a loop or xargs. This is a floor, not a vault.
set -uo pipefail
. "$(dirname "$0")/lib.sh"

SECRET='(^|[/"'"'"'=[:space:]])(\.env(\.[A-Za-z0-9_-]+)?|\.aws_credentials|\.aws/credentials|credentials\.json|clouds\.yaml|secure\.yaml|[A-Za-z0-9_.-]*openrc[A-Za-z0-9_.-]*\.sh|id_rsa|id_ed25519|[A-Za-z0-9_.-]+\.pem|[A-Za-z0-9_.-]+\.key|[A-Za-z0-9_.-]*\.tfstate(\.backup)?|[A-Za-z0-9_.-]+\.tfvars|\.npmrc)([[:space:]"'"'"';|&)]|$)'
EXAMPLE='\.(example|sample|template)([[:space:]"'"'"';|&)]|$)'
REASON="That path looks like a secret (credentials, keys, Terraform state or tfvars). Its contents would stay in the transcript. Ask the person to check the value themselves, or work from the committed .example file."

tool="$(hook_field .tool_name)"
case "$tool" in
  Read)
    p="$(hook_field .tool_input.file_path)"
    if printf '%s' "$p" | grep -Eq "$SECRET" && ! printf '%s' "$p" | grep -Eq "$EXAMPLE"; then deny "$REASON"; fi
    ;;
  Grep)
    p="$(hook_field .tool_input.path)"
    if [ -n "$p" ] && printf '%s' "$p" | grep -Eq "$SECRET" && ! printf '%s' "$p" | grep -Eq "$EXAMPLE"; then deny "$REASON"; fi
    ;;
  Bash)
    cmd="$(hook_field .tool_input.command)"
    printf '%s' "$cmd" | grep -Eq "$SECRET" || exit 0
    printf '%s' "$cmd" | grep -Eq "$EXAMPLE" && exit 0
    # A single command that only lists, tests, moves or removes the file is fine. A chain or a
    # substitution can print it through any step, so it is refused whole.
    case "$cmd" in
      *"|"*|*";"*|*"&"*|*'$('*|*'`'*|*">"*|*"<"*) deny "$REASON" ;;
    esac
    first="$(printf '%s' "$cmd" | sed -E 's/^[[:space:]]*//' | awk '{print $1}')"
    case "$first" in
      ls|stat|test|\[|rm|mv|cp|touch|chmod|mkdir|find|file|realpath|basename|dirname) exit 0 ;;
      git)
        # History and index listings name the file without printing it; `git show` and `git diff` print it.
        verb="$(printf '%s' "$cmd" | awk '{print $2}')"
        case " $cmd " in *" -p "*|*" -u "*|*" --patch "*|*" -L"*|*"--format"*|*"--pretty"*) deny "$REASON" ;; esac
        case "$verb" in log|ls-files|status|check-ignore|rm) exit 0 ;; esac ;;
    esac
    deny "$REASON"
    ;;
esac
exit 0
