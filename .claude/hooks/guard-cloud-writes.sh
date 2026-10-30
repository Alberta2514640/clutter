#!/usr/bin/env bash
# PreToolUse(Bash): a command that changes real cloud state goes to the person for approval.
# Clutter's own platform runs in a shared AWS account, and the runners act inside customers' accounts.
# A model that runs `terraform apply` or an `aws ... delete-*` from a session can break both, and some
# of those changes cannot be undone. Reads (plan, validate, describe, list, get) pass untouched.
# Asked, not refused: the person can approve the exact command in the prompt that follows.
set -uo pipefail
. "$(dirname "$0")/lib.sh"

[ "$(hook_field .tool_name)" = "Bash" ] || exit 0
cmd="$(hook_field .tool_input.command)"

# Each chained command is judged on its own words.
parts="$(split_commands "$cmd")"
while IFS= read -r part; do
  words="$(printf '%s' "$part" | sed -E 's/^[[:space:]]*([A-Za-z_][A-Za-z0-9_]*=[^[:space:]]*[[:space:]]+)*//')"
  set -- $words
  bin="${1:-}"; bin="${bin##*/}"
  case "$bin" in
    terraform|tofu)
      for w in "$@"; do
        case "$w" in
          apply|destroy|import|taint|untaint|force-unlock|push)
            ask "This runs '$bin $w', which changes real infrastructure or its state. Approve only if you meant to deploy from this session." ;;
          rm|mv|replace-provider)
            case "$words" in *" state "*) ask "This edits Terraform state ('$bin state $w'). A wrong edit orphans real resources. Approve only if intended." ;; esac ;;
        esac
      done
      ;;
    aws)
      # The service operation is the first word after the service that is not a flag.
      op=""; seen_svc=""
      shift
      for w in "$@"; do
        case "$w" in -*) continue ;; esac
        if [ -z "$seen_svc" ]; then seen_svc="$w"; continue; fi
        op="$w"; break
      done
      case "$seen_svc:$op" in
        s3:ls|s3:presign|sts:*|*:describe-*|*:list-*|*:get-*|*:wait|*:help|configure:list|configure:get|:*) ;;
        *) ask "This runs 'aws $seen_svc $op', which may change a real AWS account. Approve only if intended." ;;
      esac
      ;;
    openstack)
      for w in "$@"; do
        case "$w" in
          create|delete|set|unset|add|remove|rebuild|resize|reboot|stop|start|migrate|purge)
            ask "This runs an OpenStack '$w', which changes a real cloud project. Approve only if intended." ;;
        esac
      done
      ;;
    ansible-playbook|ansible)
      ask "Ansible acts on real hosts. Approve only if the inventory points where you expect." ;;
    build-and-push.sh)
      ask "This builds and pushes a runner image to a shared registry. Approve only if intended." ;;
    docker)
      [ "${2:-}" = "push" ] && ask "This pushes an image to a shared registry. Approve only if intended."
      ;;
  esac
done <<EOF
$parts
EOF
exit 0
