#!/usr/bin/env bash
# Drives every guard with real hook payloads and checks the decision. Run: .claude/hooks/test-hooks.sh
set -uo pipefail
H="$(cd "$(dirname "$0")" && pwd)"
pass=0; fail=0

# expect <hook> <deny|ask|allow> <payload json>
expect() {
  local out got
  out="$(printf '%s' "$3" | "$H/$1" 2>/dev/null)"
  got="$(printf '%s' "$out" | jq -r '.hookSpecificOutput.permissionDecision // "allow"' 2>/dev/null)"
  [ -z "$got" ] && got="allow"
  if [ "$got" = "$2" ]; then pass=$((pass + 1)); else fail=$((fail + 1)); echo "FAIL $1 expected $2 got $got: $3"; fi
}
bash_call() { jq -cn --arg c "$1" --arg a "${2:-}" '{tool_name:"Bash",tool_input:{command:$c}} + (if $a == "" then {} else {agent_id:$a} end)'; }
read_call() { jq -cn --arg p "$1" '{tool_name:"Read",tool_input:{file_path:$p}}'; }

# guard-secrets
expect guard-secrets.sh deny  "$(read_call /repo/.env)"
expect guard-secrets.sh deny  "$(read_call /repo/frontend/clutter/.env.local)"
expect guard-secrets.sh deny  "$(read_call /repo/.vscode/.aws_credentials)"
expect guard-secrets.sh deny  "$(read_call /repo/backend/infra/terraform.tfstate)"
expect guard-secrets.sh deny  "$(read_call /home/me/.config/openstack/clouds.yaml)"
expect guard-secrets.sh allow "$(read_call /repo/.env.example)"
expect guard-secrets.sh allow "$(read_call /repo/backend/infra/main.tf)"
expect guard-secrets.sh allow "$(read_call /repo/frontend/clutter/src/lib/env.ts)"
expect guard-secrets.sh deny  "$(bash_call 'cat .env')"
expect guard-secrets.sh deny  "$(bash_call 'ls -la && cat backend/infra/prod.tfvars')"
expect guard-secrets.sh deny  "$(bash_call 'source ./patrii-openrc.sh')"
expect guard-secrets.sh allow "$(bash_call 'ls -la .env')"
expect guard-secrets.sh allow "$(bash_call 'cp .env.example .env.local.example')"
expect guard-secrets.sh allow "$(bash_call 'go test ./...')"
expect guard-secrets.sh allow "$(bash_call 'git log --all --oneline -- .vscode/.aws_credentials')"
expect guard-secrets.sh deny  "$(bash_call 'git show ac90930:.vscode/.aws_credentials')"
expect guard-secrets.sh deny  "$(bash_call 'git log -p -- .env | cat')"
expect guard-secrets.sh deny  "$(bash_call 'git log -p -- .env')"

# guard-cloud-writes
expect guard-cloud-writes.sh ask   "$(bash_call 'cd backend/infra && terraform apply -auto-approve')"
expect guard-cloud-writes.sh ask   "$(bash_call 'terraform destroy')"
expect guard-cloud-writes.sh ask   "$(bash_call 'terraform state rm module.x')"
expect guard-cloud-writes.sh allow "$(bash_call 'terraform plan -out=p')"
expect guard-cloud-writes.sh allow "$(bash_call 'terraform validate')"
expect guard-cloud-writes.sh allow "$(bash_call 'terraform state list')"
expect guard-cloud-writes.sh ask   "$(bash_call 'aws ec2 terminate-instances --instance-ids i-1')"
expect guard-cloud-writes.sh ask   "$(bash_call 'AWS_PROFILE=x aws s3 rm s3://b/k')"
expect guard-cloud-writes.sh allow "$(bash_call 'aws sts get-caller-identity')"
expect guard-cloud-writes.sh allow "$(bash_call 'aws ec2 describe-instances --region ca-central-1')"
expect guard-cloud-writes.sh allow "$(bash_call 'aws s3 ls')"
expect guard-cloud-writes.sh ask   "$(bash_call 'openstack server delete web1')"
expect guard-cloud-writes.sh allow "$(bash_call 'openstack server list')"
expect guard-cloud-writes.sh ask   "$(bash_call 'ansible-playbook -i hosts site.yml')"
expect guard-cloud-writes.sh ask   "$(bash_call './build-and-push.sh')"
expect guard-cloud-writes.sh allow "$(bash_call 'go build ./...')"

# guard-git-writes (lanes only)
expect guard-git-writes.sh deny  "$(bash_call 'git commit -m x' lane1)"
expect guard-git-writes.sh deny  "$(bash_call 'git stash' lane1)"
expect guard-git-writes.sh deny  "$(bash_call 'git status && git checkout -- .' lane1)"
expect guard-git-writes.sh deny  "$(bash_call 'git -C /repo reset --hard' lane1)"
expect guard-git-writes.sh allow "$(bash_call 'git diff HEAD~1' lane1)"
expect guard-git-writes.sh allow "$(bash_call 'git stash list' lane1)"
expect guard-git-writes.sh allow "$(bash_call 'git log --oneline -5 | head' lane1)"
expect guard-git-writes.sh allow "$(bash_call 'git branch' lane1)"
expect guard-git-writes.sh allow "$(bash_call 'git commit -m x')"

# format-on-write: formats new files and files already formatted at HEAD; leaves CRLF or unformatted files alone.
# check <name> <condition>
check() { if eval "$2"; then pass=$((pass + 1)); else fail=$((fail + 1)); echo "FAIL format-on-write: $1"; fi; }
write_call() { jq -cn --arg p "$1" '{tool_name:"Edit",tool_input:{file_path:$p}}'; }
T="$(mktemp -d)"
git -C "$T" init -q && git -C "$T" config core.autocrlf false && git -C "$T" config user.email t@t && git -C "$T" config user.name t
printf 'package a\n\nfunc A() {}\n' > "$T/clean.go"
printf 'package a\r\n\r\nfunc B() {}\r\n' > "$T/crlf.go"
printf 'package a\n\nfunc  C() {}\n' > "$T/messy.go"
git -C "$T" add . && git -C "$T" commit -qm init
printf 'package a\n\nfunc  A() {}\n' > "$T/clean.go"
printf 'package a\r\n\r\nfunc  B() {}\r\n' > "$T/crlf.go"
printf 'package a\n\nfunc  D() {}\n' > "$T/new.go"
for f in clean crlf messy new; do write_call "$T/$f.go" | "$H/format-on-write.sh"; done
check "formats a file clean at HEAD" "grep -q '^func A' '$T/clean.go'"
check "leaves a CRLF file alone" "grep -q \$'\\r' '$T/crlf.go' && grep -q 'func  B' '$T/crlf.go'"
check "leaves a file unformatted at HEAD alone" "grep -q 'func  C' '$T/messy.go'"
check "formats a new file" "grep -q '^func D' '$T/new.go'"
rm -rf "$T"

# guard-git-writes: read-only verbs a lane needs.
expect guard-git-writes.sh allow "$(bash_call 'git check-ignore -v x' lane1)"
expect guard-git-writes.sh allow "$(bash_call 'git ls-remote --heads origin' lane1)"

# Without jq, a guard fails closed (exit 2) and the format hook still exits 0.
NOJQ="$(mktemp -d)"
for t in bash cat dirname awk printf; do p="$(command -v "$t")"; [ -n "$p" ] && ln -s "$p" "$NOJQ/$t"; done
printf '%s' "$(read_call /repo/.env)" | PATH="$NOJQ" /bin/bash "$H/guard-secrets.sh" >/dev/null 2>&1; code=$?
check "guard fails closed without jq" "[ $code -eq 2 ]"
printf '%s' "$(write_call /tmp/x.go)" | PATH="$NOJQ" /bin/bash "$H/format-on-write.sh" >/dev/null 2>&1; code=$?
check "format hook passes without jq" "[ $code -eq 0 ]"
rm -rf "$NOJQ"

echo "hooks: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
