#!/bin/bash
set -euo pipefail

# =============================================================================
# Ansible Runner Entrypoint
# Runs inside a Fargate task. Downloads playbook from S3, generates a
# pre-signed URL, then pushes execution to target instances via SSM Run
# Command. Each instance downloads the playbook and runs it locally.
# Fargate polls for completion, collects output, uploads logs, and updates
# PostgreSQL job status.
#
# Uses SSM Run Command (not SSM Session Manager) — no S3 staging bucket,
# no cross-account S3 access needed. Target EC2 instances must have SSM
# Agent running and an instance profile with AmazonSSMManagedInstanceCore.
# =============================================================================

# Required environment variables (injected by ECS task definition / run-task Lambda):
#   JOB_ID              - Unique job identifier
#   PLAYBOOK_S3_KEY     - S3 key for the playbook file
#   TARGET_INSTANCE_IDS - Comma-separated EC2 instance IDs
#   S3_BUCKET_NAME      - S3 bucket for playbooks and logs
#   PSQL_CONNECTION_STRING - PostgreSQL connection string for job status updates
#   AWS_DEFAULT_REGION  - AWS region
#   ORG_ID              - Organization ID (for org-scoped log paths)
#   PROJECT_ID          - Project ID (for org-scoped log paths)
#   DIAGRAM_ID          - Diagram ID (for org-scoped log paths)
#
# Optional:
#   EXTRA_VARS          - JSON string of extra variables for the playbook
#   PLAYBOOK_TIMEOUT    - Maximum execution time in seconds (default: 3600)

LOG_FILE="/var/log/ansible/ansible.log"

# Client role credentials (populated after sts:AssumeRole — used only for
# EC2 and SSM operations in the client account, NOT for S3 so that the
# Fargate task role remains the S3 identity and the bucket SSE exemption
# continues to apply).
CLIENT_ACCESS_KEY_ID=""
CLIENT_SECRET_ACCESS_KEY=""
CLIENT_SESSION_TOKEN=""

# Run AWS CLI calls against the client account without exporting credentials
# globally (prevents accidental reuse for S3 operations).
aws_client() {
    AWS_ACCESS_KEY_ID="$CLIENT_ACCESS_KEY_ID" \
    AWS_SECRET_ACCESS_KEY="$CLIENT_SECRET_ACCESS_KEY" \
    AWS_SESSION_TOKEN="$CLIENT_SESSION_TOKEN" \
    aws "$@"
}

log_task_role_identity() {
    local caller_identity
    if caller_identity=$(env -u AWS_ACCESS_KEY_ID -u AWS_SECRET_ACCESS_KEY -u AWS_SESSION_TOKEN \
        aws sts get-caller-identity 2>&1); then
        echo "[entrypoint] Task role caller identity: $caller_identity" >&2
    else
        echo "[entrypoint] WARNING: Failed to resolve task role caller identity: $caller_identity" >&2
    fi
}

download_playbook_base64() {
    local attempt=1
    local max_attempts=3
    local retry_delay=2
    local download_stderr=""
    local playbook_tmp=""
    local exit_code=0

    while [ $attempt -le $max_attempts ]; do
        download_stderr=$(mktemp)
        playbook_tmp=$(mktemp)

        if env -u AWS_ACCESS_KEY_ID -u AWS_SECRET_ACCESS_KEY -u AWS_SESSION_TOKEN \
            aws s3 cp "s3://${S3_BUCKET_NAME}/${PLAYBOOK_S3_KEY}" - \
            --region "$AWS_DEFAULT_REGION" >"$playbook_tmp" 2>"$download_stderr"; then
            if [ -s "$playbook_tmp" ]; then
                base64 < "$playbook_tmp" | tr -d '\n'
                rm -f "$download_stderr" "$playbook_tmp"
                return 0
            fi
            echo "[entrypoint] WARNING: Download attempt $attempt returned an empty playbook" >&2
        else
            exit_code=$?
            echo "[entrypoint] WARNING: Playbook download attempt $attempt failed (exit=$exit_code)" >&2
        fi

        if [ -s "$download_stderr" ]; then
            sed 's/^/[entrypoint] playbook download stderr: /' "$download_stderr" >&2
        fi
        rm -f "$download_stderr" "$playbook_tmp"

        log_task_role_identity

        if [ $attempt -lt $max_attempts ]; then
            echo "[entrypoint] Retrying playbook download in ${retry_delay}s..." >&2
            sleep $retry_delay
            retry_delay=$((retry_delay * 2))
        fi

        attempt=$((attempt + 1))
    done

    return 1
}

# Redact likely secrets from output and cap size before writing to logs.
sanitize_output() {
    python3 - <<'PYEOF'
import re
import sys

data = sys.stdin.read()

# Cap output size to reduce accidental large secret spill.
max_len = 16000
if len(data) > max_len:
    data = data[:max_len] + "\n...[truncated]"

# More precise patterns to avoid over-redaction
patterns = [
    # Match password/secret assignments but not in error messages
    r'(?i)(?<![A-Za-z])(password|passwd|secret|token|api[_-]?key)\s*[:=]\s*[^\s"\']+',
    # AWS access keys
    r'AKIA[0-9A-Z]{16}',
    r'ASIA[0-9A-Z]{16}',
    # PostgreSQL connection strings with credentials
    r'(?i)postgres(?:ql)?://[^:]+:[^@]+@[^\s]+'
]
for p in patterns:
    data = re.sub(p, '***REDACTED***', data)

sys.stdout.write(data)
PYEOF
}


# Retry configuration constants (global scope for use in functions and error handlers)
DOWNLOAD_PLAYBOOK_MAX_RETRIES=3
PLAYBOOK_TIMEOUT="${PLAYBOOK_TIMEOUT:-3600}"

# Create log directory and file immediately so all output is captured,
# even if Ansible never runs (e.g. early failures in inventory generation).
mkdir -p "$(dirname "$LOG_FILE")"
touch "$LOG_FILE"
# Tee all stdout and stderr into the log file while still printing to console.
exec > >(tee -a "$LOG_FILE") 2>&1

# ---- Helper: Update job status in PostgreSQL (parameterized, no SQL injection) ----
update_job_status() {
    local status="$1"
    local error_message="${2:-}"

    python3 - <<'PYEOF' "$JOB_ID" "$status" "$error_message"
import sys
import os
import psycopg2
from datetime import datetime, timezone

job_id, status, error_message = sys.argv[1], sys.argv[2], sys.argv[3]
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
conn_string = os.environ.get('PSQL_CONNECTION_STRING', '')

if not conn_string:
    print("[update_job_status] WARNING: PSQL_CONNECTION_STRING is missing", file=sys.stderr)
    sys.exit(0)

try:
    with psycopg2.connect(conn_string) as conn:
        with conn.cursor() as cur:
            if error_message:
                cur.execute(
                    "UPDATE jobs SET status = %s, error_message = %s, updated_at = %s WHERE id = %s",
                    (status, error_message, now, job_id)
                )
            else:
                cur.execute(
                    "UPDATE jobs SET status = %s, updated_at = %s WHERE id = %s",
                    (status, now, job_id)
                )
            conn.commit()
except Exception as e:
    print(f"[update_job_status] WARNING: Failed to update job status: {e}", file=sys.stderr)
PYEOF
}

# ---- Helper: Upload logs to S3 and record the key in the database ----
upload_logs() {
    local upload_failed=0
    local max_retries=3
    local retry_count=0
    local retry_delay=2

    if [ -f "$LOG_FILE" ]; then
        local log_key="${ORG_ID:-unknown}/${PROJECT_ID:-unknown}/${DIAGRAM_ID:-unknown}/playbooks/logs/${JOB_ID:-unknown}.log"

        # Retry loop for S3 upload with exponential backoff.
        # Use Fargate task role by unsetting assumed role credentials contextually using env.
        while [ $retry_count -lt $max_retries ]; do
            if env -u AWS_ACCESS_KEY_ID -u AWS_SECRET_ACCESS_KEY -u AWS_SESSION_TOKEN \
                aws s3 cp "$LOG_FILE" "s3://${S3_BUCKET_NAME:-unknown}/${log_key}" \
                --region "$AWS_DEFAULT_REGION" \
                --sse AES256 2>&1; then
                echo "[upload_logs] Log file uploaded successfully to s3://${S3_BUCKET_NAME}/${log_key}"

                # Record log location in database (with retry)
                local db_retry_count=0
                local db_max_retries=3
                local db_update_success=0
                while [ $db_retry_count -lt $db_max_retries ]; do
                    if python3 - <<'PYEOF' "$JOB_ID" "$log_key"
import sys
import os
import psycopg2
from datetime import datetime, timezone

job_id, log_key = sys.argv[1], sys.argv[2]
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
conn_string = os.environ.get('PSQL_CONNECTION_STRING', '')

if not conn_string:
    print("[upload_logs] WARNING: PSQL_CONNECTION_STRING is missing", file=sys.stderr)
    sys.exit(1)

try:
    with psycopg2.connect(conn_string) as conn:
        with conn.cursor() as cur:
            cur.execute(
                "UPDATE jobs SET log_s3_key = %s, updated_at = %s WHERE id = %s",
                (log_key, now, job_id)
            )
            conn.commit()
    print(f"[upload_logs] Log S3 key recorded in database")
except Exception as e:
    print(f"[upload_logs] WARNING: Failed to record log_s3_key in database: {e}", file=sys.stderr)
    sys.exit(1)
PYEOF
                    then
                        db_update_success=1
                        break
                    else
                        db_retry_count=$((db_retry_count + 1))
                        if [ $db_retry_count -lt $db_max_retries ]; then
                            echo "[upload_logs] Database update failed, retrying in $((db_retry_count * 2))s..."
                            sleep $((db_retry_count * 2))
                        fi
                    fi
                done
                # If DB update retries were exhausted, log error and return failure
                if [ $db_update_success -eq 0 ]; then
                    echo "[upload_logs] ERROR: Failed to update database with log_s3_key after $db_max_retries attempts" >&2
                    return 1
                fi
                return 0
            else
                retry_count=$((retry_count + 1))
                if [ $retry_count -lt $max_retries ]; then
                    echo "[upload_logs] WARNING: S3 upload failed, retrying in ${retry_delay}s (attempt $retry_count/$max_retries)" >&2
                    sleep $retry_delay
                    retry_delay=$((retry_delay * 2))
                else
                    echo "[upload_logs] ERROR: S3 upload failed after $max_retries attempts" >&2
                    upload_failed=1
                fi
            fi
        done
    fi
    return $upload_failed
}

# ---- Trap: Handle signals for graceful shutdown ----
cleanup() {
    echo "[entrypoint] Received shutdown signal. Uploading logs."
    update_job_status "FAILED" "Task was terminated by signal"
    upload_logs || true
    exit 1
}
trap cleanup SIGTERM SIGINT

# ---- Validate required env vars ----
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-${AWS_REGION:-}}"
for var in JOB_ID PLAYBOOK_S3_KEY TARGET_INSTANCE_IDS S3_BUCKET_NAME PSQL_CONNECTION_STRING AWS_DEFAULT_REGION ORG_ID PROJECT_ID DIAGRAM_ID CLIENT_ROLE_ARN ASSUME_ROLE_EXTERNAL_ID; do
    if [ -z "${!var:-}" ]; then
        echo "[entrypoint] ERROR: Missing required environment variable: $var"
        upload_logs || true
        update_job_status "FAILED" "Missing environment variable: $var"
        exit 1
    fi
done

# Validate timeout after required env vars and helper functions are available.
if ! [[ "$PLAYBOOK_TIMEOUT" =~ ^[0-9]+$ ]] || [ "$PLAYBOOK_TIMEOUT" -lt 60 ] || [ "$PLAYBOOK_TIMEOUT" -gt 7200 ]; then
    echo "[entrypoint] ERROR: PLAYBOOK_TIMEOUT must be an integer between 60 and 7200, got: $PLAYBOOK_TIMEOUT"
    update_job_status "FAILED" "Invalid PLAYBOOK_TIMEOUT value"
    upload_logs || true
    exit 1
fi

# Validate TARGET_INSTANCE_IDS format to prevent injection attacks
if [[ ! "$TARGET_INSTANCE_IDS" =~ ^i-[0-9a-fA-F]{8,17}(,i-[0-9a-fA-F]{8,17})*$ ]]; then
    echo "[entrypoint] ERROR: Invalid instance ID format in TARGET_INSTANCE_IDS"
    upload_logs || true
    update_job_status "FAILED" "Invalid EC2 instance ID format"
    exit 1
fi

echo "[entrypoint] Starting Ansible job: $JOB_ID"
echo "[entrypoint] Playbook: s3://${S3_BUCKET_NAME}/${PLAYBOOK_S3_KEY}"
echo "[entrypoint] Targets: $TARGET_INSTANCE_IDS"
echo "[entrypoint] Connection: SSM Run Command (local execution on targets)"

# 1. Mark job as RUNNING
update_job_status "RUNNING"

# 2. Assume customer role
# Credentials are stored in variables — NOT exported to the environment so
# that all S3 operations (playbook download, log upload, SSM plugin file
# staging) continue to use the Fargate task-role identity, which is already
# exempt from the bucket's SSE enforcement policy.
echo "[entrypoint] Assuming customer role: $CLIENT_ROLE_ARN"
if ! CREDS=$(aws sts assume-role \
  --role-arn "$CLIENT_ROLE_ARN" \
  --role-session-name "clutter-ansible-${JOB_ID}" \
  --external-id "$ASSUME_ROLE_EXTERNAL_ID" 2>&1); then
    echo "[entrypoint] ERROR: Failed to assume client role"
    upload_logs || true
    update_job_status "FAILED" "Failed to assume client AWS role"
    exit 1
fi

# Validate credentials response before parsing to prevent exposure of error messages
if ! echo "$CREDS" | python3 -c 'import sys, json; d=json.load(sys.stdin); sys.exit(0 if "Credentials" in d else 1)' 2>&1; then
    echo "[entrypoint] ERROR: Invalid credentials response from STS"
    upload_logs || true
    update_job_status "FAILED" "Invalid STS credentials response"
    exit 1
fi

CLIENT_ACCESS_KEY_ID=$(echo "$CREDS" | python3 -c 'import sys, json; print(json.load(sys.stdin)["Credentials"]["AccessKeyId"])')
CLIENT_SECRET_ACCESS_KEY=$(echo "$CREDS" | python3 -c 'import sys, json; print(json.load(sys.stdin)["Credentials"]["SecretAccessKey"])')
CLIENT_SESSION_TOKEN=$(echo "$CREDS" | python3 -c 'import sys, json; print(json.load(sys.stdin)["Credentials"]["SessionToken"])')

# 3. Download playbook with the Fargate task role and inline it in the SSM payload
#    so target instances do not need direct S3 access.
echo "[entrypoint] Downloading playbook from S3..."

if ! PLAYBOOK_B64=$(download_playbook_base64); then
    echo "[entrypoint] ERROR: Failed to download playbook from S3 after retries"
    update_job_status "FAILED" "Failed to download playbook from S3"
    upload_logs || true
    exit 1
fi

echo "[entrypoint] Playbook downloaded and encoded for transport"

# 4. Prepare extra vars for remote execution without shell interpolation.
#    We pass base64 and materialize a temp file on the target instance.
EXTRA_VARS_B64=""
if [ -n "${EXTRA_VARS:-}" ] && [ "$EXTRA_VARS" != "null" ] && [ "$EXTRA_VARS" != "{}" ]; then
    if ! echo "$EXTRA_VARS" | python3 -c 'import sys,json; json.load(sys.stdin)' >/dev/null 2>&1; then
        echo "[entrypoint] ERROR: EXTRA_VARS must be valid JSON"
        update_job_status "FAILED" "Invalid EXTRA_VARS JSON"
        upload_logs || true
        exit 1
    fi

    # Block likely secret values from being sent in SSM command payload.
    if ! python3 - "$EXTRA_VARS" <<'PYEOF'
import json
import re
import sys

try:
    obj = json.loads(sys.argv[1])
except Exception:
    print("invalid json")
    sys.exit(1)

secret_key_re = re.compile(r'(password|passwd|secret|token|api[_-]?key|access[_-]?key)', re.I)

def has_sensitive(d):
    if isinstance(d, dict):
        for k, v in d.items():
            if secret_key_re.search(str(k)) and v not in (None, "", {}, []):
                return True
            if has_sensitive(v):
                return True
    elif isinstance(d, list):
        return any(has_sensitive(x) for x in d)
    return False

if has_sensitive(obj):
    print("sensitive keys present")
    sys.exit(1)

sys.exit(0)
PYEOF
    then
        echo "[entrypoint] ERROR: EXTRA_VARS contains sensitive keys and cannot be sent via SSM command payload"
        update_job_status "FAILED" "EXTRA_VARS contains sensitive keys; use non-sensitive vars only"
        upload_logs || true
        exit 1
    fi

    EXTRA_VARS_B64=$(printf '%s' "$EXTRA_VARS" | base64 | tr -d '\n')
fi

# 5. Build the shell script that each target instance will execute via SSM Run Command.
#    The script: installs Ansible if missing, materializes the playbook from an
#    inline base64 payload, then runs it locally with --connection=local.
read -r -d '' SSM_SCRIPT << 'SSMEOF' || true
#!/bin/bash
set -e

echo "=== Clutter Ansible Runner ==="
echo "Instance: $(curl -sf http://169.254.169.254/latest/meta-data/instance-id 2>/dev/null || hostname)"
echo "Date: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"

# Install Ansible if not present
if ! command -v ansible-playbook &>/dev/null; then
    echo "Ansible not found, installing..."
    if command -v pip3 &>/dev/null; then
        pip3 install ansible --quiet 2>&1
    elif command -v yum &>/dev/null; then
        yum install -y ansible 2>&1
    elif command -v apt-get &>/dev/null; then
        apt-get update -qq && apt-get install -y ansible 2>&1
    else
        echo "ERROR: Cannot install Ansible - no supported package manager found"
        exit 1
    fi
    echo "Ansible installed successfully"
fi

echo "Ansible version: $(ansible-playbook --version | head -1)"

# Materialize playbook from inline base64 payload
mkdir -p /tmp/clutter-ansible
echo "Writing playbook payload to local file..."
if ! (printf '%s' "PLAYBOOK_B64_PLACEHOLDER" | base64 -d > /tmp/clutter-ansible/playbook.yml 2>/dev/null || \
      printf '%s' "PLAYBOOK_B64_PLACEHOLDER" | base64 --decode > /tmp/clutter-ansible/playbook.yml 2>/dev/null); then
    echo "ERROR: Failed to decode playbook payload"
    exit 1
fi
if [ ! -s /tmp/clutter-ansible/playbook.yml ]; then
    echo "ERROR: Playbook file is empty after decode"
    exit 1
fi
echo "Playbook materialized successfully"

# Create local inventory (playbooks must use hosts: all or hosts: localhost)
echo "localhost ansible_connection=local" > /tmp/clutter-ansible/inventory

# Run playbook locally
cd /tmp/clutter-ansible
if [ -n "EXTRA_VARS_B64_PLACEHOLDER" ]; then
    if ! (printf '%s' "EXTRA_VARS_B64_PLACEHOLDER" | base64 -d > /tmp/clutter-ansible/extra_vars.json 2>/dev/null || \
          printf '%s' "EXTRA_VARS_B64_PLACEHOLDER" | base64 --decode > /tmp/clutter-ansible/extra_vars.json 2>/dev/null); then
        echo "ERROR: Failed to decode EXTRA_VARS payload"
        exit 1
    fi
    chmod 600 /tmp/clutter-ansible/extra_vars.json
    ansible-playbook -i inventory playbook.yml --extra-vars @/tmp/clutter-ansible/extra_vars.json -v
else
    ansible-playbook -i inventory playbook.yml -v
fi

echo "=== Playbook execution complete ==="
SSMEOF

# Inject actual values into the script template
SSM_SCRIPT="${SSM_SCRIPT//PLAYBOOK_B64_PLACEHOLDER/$PLAYBOOK_B64}"
SSM_SCRIPT="${SSM_SCRIPT//EXTRA_VARS_B64_PLACEHOLDER/$EXTRA_VARS_B64}"

# 6. Send SSM Run Command to all target instances
echo "[entrypoint] Sending SSM Run Command to instances: $TARGET_INSTANCE_IDS"

# Parse comma-separated instance IDs into array for SSM CLI
IFS=',' read -ra INSTANCE_ARRAY <<< "$TARGET_INSTANCE_IDS"

# Convert the script to a JSON-safe string for the SSM commands parameter
SSM_COMMANDS_JSON=$(echo "$SSM_SCRIPT" | python3 -c 'import sys,json; print(json.dumps([sys.stdin.read()]))')

set +e
COMMAND_ID=$(aws_client ssm send-command \
    --document-name "AWS-RunShellScript" \
    --instance-ids "${INSTANCE_ARRAY[@]}" \
    --parameters "{\"commands\":$SSM_COMMANDS_JSON}" \
    --timeout-seconds "$PLAYBOOK_TIMEOUT" \
    --region "$AWS_DEFAULT_REGION" \
    --query "Command.CommandId" \
    --output text 2>&1)
SEND_EXIT=$?
set -e

if [ $SEND_EXIT -ne 0 ] || [ -z "$COMMAND_ID" ]; then
    echo "[entrypoint] ERROR: Failed to send SSM command (exit=$SEND_EXIT): $COMMAND_ID"
    update_job_status "FAILED" "Failed to send SSM Run Command"
    upload_logs || true
    exit 1
fi

echo "[entrypoint] SSM Command ID: $COMMAND_ID"

# 7. Poll for completion
echo "[entrypoint] Waiting for command to complete on all instances..."
POLL_INTERVAL=10
MAX_WAIT="$PLAYBOOK_TIMEOUT"
ELAPSED=0
ALL_DONE=false

while [ "$ELAPSED" -lt "$MAX_WAIT" ] && [ "$ALL_DONE" = "false" ]; do
    sleep "$POLL_INTERVAL"
    ELAPSED=$((ELAPSED + POLL_INTERVAL))

    STATUS=$(aws_client ssm list-commands \
        --command-id "$COMMAND_ID" \
        --region "$AWS_DEFAULT_REGION" \
        --query "Commands[0].Status" \
        --output text 2>&1) || STATUS="Error"

    echo "[entrypoint] Command status: $STATUS (${ELAPSED}s elapsed)"

    case "$STATUS" in
        Success|Failed|Cancelled|TimedOut)
            ALL_DONE=true
            ;;
        Pending|InProgress)
            ;;
        *)
            echo "[entrypoint] WARNING: Unexpected status: $STATUS"
            ;;
    esac
done

# 8. Collect output from each instance
echo "[entrypoint] Collecting output from instances..."
OVERALL_SUCCESS=true

for INSTANCE_ID in "${INSTANCE_ARRAY[@]}"; do
    echo ""
    echo "========== Instance: $INSTANCE_ID =========="

    set +e
    INVOCATION=$(aws_client ssm get-command-invocation \
        --command-id "$COMMAND_ID" \
        --instance-id "$INSTANCE_ID" \
        --region "$AWS_DEFAULT_REGION" \
        --output json 2>&1)
    set -e

    INST_STATUS=$(echo "$INVOCATION" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("Status","Unknown"))' 2>/dev/null || echo "Unknown")
    STDOUT=$(echo "$INVOCATION" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("StandardOutputContent",""))' 2>/dev/null || echo "")
    STDERR=$(echo "$INVOCATION" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("StandardErrorContent",""))' 2>/dev/null || echo "")

    echo "Status: $INST_STATUS"
    if [ -n "$STDOUT" ]; then
        echo "--- STDOUT ---"
        echo "$STDOUT" | sanitize_output
    fi
    if [ -n "$STDERR" ]; then
        echo "--- STDERR ---"
        echo "$STDERR" | sanitize_output
    fi
    echo "========== End: $INSTANCE_ID =========="

    if [ "$INST_STATUS" != "Success" ]; then
        OVERALL_SUCCESS=false
    fi
done

# 9. Final status
if [ "$ALL_DONE" = "false" ]; then
    echo "[entrypoint] ERROR: Command timed out after ${MAX_WAIT}s"
    update_job_status "FAILED" "SSM command timed out after ${MAX_WAIT}s"
    upload_logs || true
    exit 1
elif [ "$OVERALL_SUCCESS" = "true" ]; then
    echo "[entrypoint] All instances completed successfully"
    update_job_status "COMPLETED"
    upload_logs || true
    exit 0
else
    echo "[entrypoint] One or more instances failed"
    update_job_status "FAILED" "Playbook failed on one or more instances"
    upload_logs || true
    exit 1
fi
