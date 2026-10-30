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

echo "hooks: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
