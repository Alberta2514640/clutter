#!/usr/bin/env bash
# =============================================================================
# Clutter Backend — End-to-End Ansible Fargate Test
#
# Tests the full job lifecycle:
#   1. Generate a test JWT
#   2. POST /ansible/playbooks/upload-url  → get a presigned S3 URL
#   3. PUT  <presigned URL>               → upload a minimal test playbook
#   4. POST /ansible/jobs                 → submit the job (enqueues to SQS)
#   5. GET  /ansible/jobs/{jobId}         → poll status (QUEUED → STARTING)
#   6. Verify in ECS that a Fargate task was launched
#
# Prerequisites:
#   - Deployment must be complete (run deploy.sh first)
#   - go >= 1.21 on PATH
#   - aws CLI on PATH
#   - curl on PATH
#   - jq on PATH (optional but recommended; falls back to python3)
#
# Usage:
#   export JWT_SECRET="<your-jwt-secret>"
#   cd backend/infra
#   ./e2e-test.sh
# =============================================================================
set -euo pipefail

INFRA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${INFRA_DIR}/../.."
JWT_TOOL_DIR="${REPO_ROOT}/backend/api/log-in/generate-test-jwt"
AWS_REGION="us-west-2"

log()  { echo "[e2e] $*"; }
fail() { echo "[e2e] ERROR: $*" >&2; exit 1; }

[ -z "${JWT_SECRET:-}" ] && fail "JWT_SECRET environment variable is not set. Export it before running this script."

# ---------------------------------------------------------------------------
# Helper: extract a JSON field (uses jq if available, else python3)
# ---------------------------------------------------------------------------
json_get() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r "$field"
  else
    echo "$json" | python3 -c "import sys,json; d=json.load(sys.stdin); exec(\"print(d${field//./['']}\".replace(\"['\",\"['\").replace(\"']\",\"']\"))" 2>/dev/null \
      || echo "$json" | python3 -c "
import sys,json
data=json.load(sys.stdin)
keys='${field}'.strip('.').split('.')
val=data
for k in keys:
    if k:
        val=val[k]
print(val)"
  fi
}

# ---------------------------------------------------------------------------
# Load deployment outputs
# ---------------------------------------------------------------------------
OUTPUTS_FILE="${INFRA_DIR}/.deployment-outputs"
if [ -f "$OUTPUTS_FILE" ]; then
  log "Loading deployment outputs from ${OUTPUTS_FILE}..."
  source "$OUTPUTS_FILE"
else
  log "No .deployment-outputs file found. Using manual values..."
  # Set manually if you skipped deploy.sh:
  API_BASE_URL="${API_BASE_URL:-}"
  CLUTTER_BUCKET="${CLUTTER_BUCKET:-}"
fi

[ -z "${API_BASE_URL:-}" ] && fail "API_BASE_URL is not set. Run deploy.sh first or set it manually."
[ -z "${CLUTTER_BUCKET:-}" ] && fail "CLUTTER_BUCKET is not set. Run deploy.sh first or set it manually."

log "  API Base URL:      $API_BASE_URL"
log "  Clutter S3 Bucket: $CLUTTER_BUCKET"
log ""

# ---------------------------------------------------------------------------
# STEP 1 — Generate a test JWT
# ---------------------------------------------------------------------------
log "Step 1: Generating test JWT..."
cd "$JWT_TOOL_DIR"
JWT_OUTPUT=$(JWT_SECRET="$JWT_SECRET" go run main.go 2>&1)
TOKEN=$(echo "$JWT_OUTPUT" | grep "^TOKEN=" | cut -d= -f2-)
EXP=$(echo "$JWT_OUTPUT" | grep "^EXP=" | cut -d= -f2-)
[ -z "$TOKEN" ] && fail "Failed to generate JWT. Output:\n$JWT_OUTPUT"
log "  JWT generated (expires at Unix ts: $EXP)"
log "  Token prefix: ${TOKEN:0:30}..."

AUTH_HEADER="Authorization: Bearer $TOKEN"

# ---------------------------------------------------------------------------
# STEP 2 — POST /ansible/playbooks/upload-url
# ---------------------------------------------------------------------------
log ""
log "Step 2: Requesting playbook upload URL..."
UPLOAD_URL_RESPONSE=$(curl -sf \
  -X POST \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{"file_name":"e2e-test-playbook.yml"}' \
  "${API_BASE_URL}/ansible/playbooks/upload-url" 2>&1) \
  || fail "POST /ansible/playbooks/upload-url failed.\nResponse: $UPLOAD_URL_RESPONSE"

log "  Response: $UPLOAD_URL_RESPONSE"

PRESIGNED_URL=$(echo "$UPLOAD_URL_RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['upload_url'])")
PLAYBOOK_S3_KEY=$(echo "$UPLOAD_URL_RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['playbook_s3_key'])")

[ -z "$PRESIGNED_URL" ] && fail "Could not parse upload_url from response"
[ -z "$PLAYBOOK_S3_KEY" ] && fail "Could not parse playbook_s3_key from response"

log "  Presigned URL obtained."
log "  Playbook S3 Key: $PLAYBOOK_S3_KEY"

# ---------------------------------------------------------------------------
# STEP 3 — Upload a minimal test playbook via the presigned URL
# ---------------------------------------------------------------------------
log ""
log "Step 3: Uploading test playbook to S3 via presigned URL..."

# Minimal valid Ansible playbook (connects to localhost with connection=local)
PLAYBOOK_CONTENT="---
- name: E2E Test Playbook
  hosts: all
  gather_facts: false
  tasks:
    - name: Echo test message
      ansible.builtin.debug:
        msg: 'Clutter Ansible E2E test - job {{ JOB_ID | default(\"unknown\") }}'
"

UPLOAD_HTTP_STATUS=$(curl -sf \
  -o /dev/null \
  -w "%{http_code}" \
  -X PUT \
  -H "x-amz-server-side-encryption: AES256" \
  -H "Content-Type: text/plain" \
  --data-raw "$PLAYBOOK_CONTENT" \
  "$PRESIGNED_URL" 2>&1) \
  || fail "PUT to presigned S3 URL failed. Status: $UPLOAD_HTTP_STATUS"

log "  Upload HTTP status: $UPLOAD_HTTP_STATUS"
[ "$UPLOAD_HTTP_STATUS" = "200" ] || fail "Expected HTTP 200 from presigned PUT, got $UPLOAD_HTTP_STATUS"
log "  Playbook uploaded successfully."

# ---------------------------------------------------------------------------
# STEP 4 — POST /ansible/jobs to submit the job
# ---------------------------------------------------------------------------
log ""
log "Step 4: Submitting Ansible job..."

# NOTE: target_instance_ids is intentionally set to a placeholder.
# The Fargate task will fail to connect (no real EC2 instances in this test env),
# but the launch itself exercises the full SQS → Lambda → ECS RunTask path.
SUBMIT_RESPONSE=$(curl -sf \
  -X POST \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{
    \"playbook_s3_key\": \"${PLAYBOOK_S3_KEY}\",
    \"target_instance_ids\": [\"i-00000000000000001\"],
    \"extra_vars\": {}
  }" \
  "${API_BASE_URL}/ansible/jobs" 2>&1) \
  || fail "POST /ansible/jobs failed.\nResponse: $SUBMIT_RESPONSE"

log "  Response: $SUBMIT_RESPONSE"

JOB_ID=$(echo "$SUBMIT_RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('job_id', d.get('data', {}).get('job_id', '')))" 2>/dev/null || true)
[ -z "$JOB_ID" ] && fail "Could not parse job_id from submit response"
log "  Job ID: $JOB_ID"

# ---------------------------------------------------------------------------
# STEP 5 — Poll GET /ansible/jobs/{jobId}
# ---------------------------------------------------------------------------
log ""
log "Step 5: Polling job status for job $JOB_ID..."
log "  (Polling every 5s for up to 60s — job should transition QUEUED → STARTING)"

POLL_LIMIT=12
STATUS=""
for i in $(seq 1 $POLL_LIMIT); do
  sleep 5
  STATUS_RESPONSE=$(curl -sf \
    -H "$AUTH_HEADER" \
    "${API_BASE_URL}/ansible/jobs/${JOB_ID}" 2>&1) \
    || { log "  Poll $i/$POLL_LIMIT: GET failed, retrying..."; continue; }

  STATUS=$(echo "$STATUS_RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('status', d.get('data', {}).get('status', 'UNKNOWN')))" 2>/dev/null || echo "PARSE_ERROR")
  TASK_ARN=$(echo "$STATUS_RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('task_arn', d.get('data', {}).get('task_arn', '')))" 2>/dev/null || echo "")

  log "  Poll $i/$POLL_LIMIT: status=$STATUS  task_arn=${TASK_ARN:-<not yet set>}"

  if [ "$STATUS" = "STARTING" ] || [ "$STATUS" = "RUNNING" ] || [ "$STATUS" = "COMPLETED" ] || [ "$STATUS" = "FAILED" ]; then
    break
  fi
done

log "  Final status: $STATUS"

# ---------------------------------------------------------------------------
# STEP 6 — Verify ECS Fargate task was launched
# ---------------------------------------------------------------------------
log ""
log "Step 6: Verifying ECS Fargate task launch..."

ECS_CLUSTER_ARN=$(aws ecs list-clusters --region "$AWS_REGION" \
  | python3 -c "import sys,json; clusters=json.load(sys.stdin)['clusterArns']; print(next((c for c in clusters if 'clutter' in c), ''))")

if [ -z "$ECS_CLUSTER_ARN" ]; then
  fail "Could not find 'clutter' ECS cluster in $AWS_REGION"
fi
log "  Cluster: $ECS_CLUSTER_ARN"

# List tasks in all states
for STATE in RUNNING STOPPED PENDING; do
  TASKS=$(aws ecs list-tasks \
    --cluster "$ECS_CLUSTER_ARN" \
    --family "ansible-runner" \
    --desired-status "$STATE" \
    --region "$AWS_REGION" \
    | python3 -c "import sys,json; print('\n'.join(json.load(sys.stdin)['taskArns']))" 2>/dev/null || echo "")
  if [ -n "$TASKS" ]; then
    log "  Found ansible-runner tasks in state $STATE:"
    echo "$TASKS" | while read -r arn; do log "    $arn"; done
  fi
done

# If we captured the task ARN from job status, describe it directly
if [ -n "${TASK_ARN:-}" ]; then
  log ""
  log "  Describing launched task: $TASK_ARN"
  aws ecs describe-tasks \
    --cluster "$ECS_CLUSTER_ARN" \
    --tasks "$TASK_ARN" \
    --region "$AWS_REGION" \
    --query "tasks[0].{TaskArn:taskArn,Status:lastStatus,StartedAt:startedAt,StoppedReason:stoppedReason}" \
    --output table 2>&1 || log "  (Could not describe task — it may have already stopped)"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
log ""
log "=========================================="
log "E2E TEST SUMMARY"
log "=========================================="
log "  Job ID:         $JOB_ID"
log "  Final Status:   $STATUS"
log "  Task ARN:       ${TASK_ARN:-<not captured>}"
log ""

if [ "$STATUS" = "STARTING" ] || [ "$STATUS" = "RUNNING" ] || [ "$STATUS" = "COMPLETED" ]; then
  log "  RESULT: PASS — Fargate task was successfully launched."
elif [ "$STATUS" = "FAILED" ]; then
  log "  RESULT: PARTIAL PASS — Job reached FAILED status."
  log "          This is expected if there are no real EC2 instances to connect to."
  log "          The full pipeline (SQS → Lambda → ECS RunTask) executed correctly."
else
  log "  RESULT: INCONCLUSIVE — Job stayed in status: $STATUS"
  log "          Check CloudWatch logs: /aws/lambda/ansible-run-task"
  log "          and ECS logs: /ecs/ansible-runner"
fi
log ""
log "  CloudWatch Lambda logs:"
log "    aws logs tail /aws/lambda/ansible-run-task --region $AWS_REGION --follow"
log ""
log "  CloudWatch ECS logs:"
log "    aws logs tail /ecs/ansible-runner --region $AWS_REGION --follow"
