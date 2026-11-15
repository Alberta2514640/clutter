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
  # Wrappers that run another command (env -i, nohup, timeout 5, sudo, ...) hide the real one:
  # step past them, and past env's flags and VAR=value words, to the command that runs.
  while [ $# -gt 0 ]; do
    case "${1##*/}" in
      env)
        shift
        while [ $# -gt 0 ]; do
          case "$1" in
            -u|--unset|-C|--chdir|-S|--split-string) shift 2 ;;
            -*|*=*) shift ;;
            *) break ;;
          esac
        done ;;
      command|exec|nohup|time|sudo|stdbuf|nice|caffeinate)
        shift
        while [ $# -gt 0 ]; do case "$1" in -n) shift 2 ;; -*) shift ;; *) break ;; esac; done ;;
      timeout)
        shift
        while [ $# -gt 0 ]; do case "$1" in -*) shift ;; *) break ;; esac; done
        [ $# -gt 0 ] && shift ;;  # the duration
      *) break ;;
    esac
  done
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

# Backstop for commands run inside another one (bash -c '...', limactl shell vm -- ..., ssh host ...):
# if the whole text names a cloud-changing command anywhere, ask.
flat="$(printf '%s' "$cmd" | tr '\n' ' ')"
if printf '%s' "$flat" | grep -Eq '(^|[^[:alnum:]_.-])(terraform|tofu)[[:space:]]([^;&|]*[[:space:]])?(apply|destroy|import|taint|untaint|force-unlock)([[:space:]]|$|["'"'"'])'; then
  ask "This runs Terraform apply, destroy or a state change inside another command. Approve only if you meant to deploy from this session."
fi
if printf '%s' "$flat" | grep -Eq '(^|[^[:alnum:]_.-])openstack[[:space:]]([^;&|]*[[:space:]])?(create|delete|set|unset|add|remove|rebuild|resize|reboot|stop|start|migrate|purge)([[:space:]]|$|["'"'"'])'; then
  ask "This runs an OpenStack command that changes a real cloud project, inside another command. Approve only if intended."
fi
if printf '%s' "$flat" | grep -Eq '(^|[^[:alnum:]_.-])ansible-playbook([[:space:]]|$)'; then
  ask "Ansible acts on real hosts. Approve only if the inventory points where you expect."
fi
exit 0
