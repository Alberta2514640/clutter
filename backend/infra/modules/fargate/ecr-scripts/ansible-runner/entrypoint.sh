#!/bin/bash
set -euo pipefail

# =============================================================================
# Ansible Runner Entrypoint
# Runs inside a Fargate task. Downloads playbook from S3, generates inventory,
# executes ansible-playbook, uploads logs, and updates PostgreSQL job status.
#
# Uses SSM Session Manager for connectivity — no SSH keys or port 22 needed.
# Target EC2 instances must have SSM Agent running and an instance profile
# with AmazonSSMManagedInstanceCore policy.
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
INVENTORY_FILE="/playbooks/inventory.yml"

# Client role credentials (populated after sts:AssumeRole — used only for
# EC2 and SSM operations in the client account, NOT for S3 so that the
# Fargate task role remains the S3 identity and the bucket SSE exemption
# continues to apply).
CLIENT_ACCESS_KEY_ID=""
CLIENT_SECRET_ACCESS_KEY=""
CLIENT_SESSION_TOKEN=""

# Retry configuration constants (global scope for use in functions and error handlers)
DOWNLOAD_PLAYBOOK_MAX_RETRIES=3
GENERATE_INVENTORY_MAX_RETRIES=3
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
        local log_key="${ORG_ID:-unknown}/${PROJECT_ID:-unknown}/${DIAGRAM_ID:-unknown}/logs/${JOB_ID:-unknown}/ansible.log"

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
stop_ec2_instances() {
    if [ -n "${TARGET_INSTANCE_IDS:-}" ] && [ -n "${CLIENT_ACCESS_KEY_ID:-}" ]; then
        local id_array
        IFS=',' read -r -a id_array <<< "$TARGET_INSTANCE_IDS"
        echo "[entrypoint] Stopping EC2 instances: ${id_array[*]}"
        AWS_ACCESS_KEY_ID="$CLIENT_ACCESS_KEY_ID" \
        AWS_SECRET_ACCESS_KEY="$CLIENT_SECRET_ACCESS_KEY" \
        AWS_SESSION_TOKEN="$CLIENT_SESSION_TOKEN" \
        aws ec2 stop-instances --instance-ids "${id_array[@]}" --region "$AWS_DEFAULT_REGION" 2>&1 || \
            echo "[entrypoint] WARNING: Failed to stop EC2 instances (non-fatal)"
    fi
}

cleanup() {
    echo "[entrypoint] Received shutdown signal. Stopping EC2 instances and uploading logs."
    stop_ec2_instances
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
echo "[entrypoint] Connection: SSM Session Manager"

# 1. Mark job as RUNNING
update_job_status "RUNNING"

# 2. Download playbook from S3
echo "[entrypoint] Downloading playbook from S3..."
download_playbook_with_retry() {
    local retry_count=0
    local retry_delay=5

    while [ $retry_count -lt $DOWNLOAD_PLAYBOOK_MAX_RETRIES ]; do
        if aws s3 cp "s3://${S3_BUCKET_NAME}/${PLAYBOOK_S3_KEY}" /playbooks/main.yml --region "$AWS_DEFAULT_REGION" 2>&1; then
            return 0
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $DOWNLOAD_PLAYBOOK_MAX_RETRIES ]; then
                echo "[entrypoint] WARNING: Playbook download failed, retrying in ${retry_delay}s (attempt $retry_count/$DOWNLOAD_PLAYBOOK_MAX_RETRIES)"
                sleep $retry_delay
                retry_delay=$((retry_delay * 2))
            else
                return 1
            fi
        fi
    done
}

if ! download_playbook_with_retry; then
    echo "[entrypoint] ERROR: Failed to download playbook after $DOWNLOAD_PLAYBOOK_MAX_RETRIES attempts"
    upload_logs || true
    update_job_status "FAILED" "Failed to download playbook from S3"
    exit 1
fi

# Assume customer role
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

# 3. Generate ansible.cfg (configured for SSM, no SSH)
cat > /playbooks/ansible.cfg << 'ANSIBLECFG'
[defaults]
inventory = /playbooks/inventory.yml
host_key_checking = False
log_path = /var/log/ansible/ansible.log
timeout = 60
forks = 10
stdout_callback = default
remote_tmp = /tmp/.ansible/tmp
allow_unsafe_lookups = False

[privilege_escalation]
become = True
become_method = sudo
ANSIBLECFG

export ANSIBLE_CONFIG=/playbooks/ansible.cfg

# 4. Generate dynamic inventory from instance IDs (uses SSM connection)
echo "[entrypoint] Generating inventory..."
generate_inventory_with_retry() {
    local retry_count=0
    local retry_delay=5

    while [ $retry_count -lt $GENERATE_INVENTORY_MAX_RETRIES ]; do
        if python3 /usr/local/bin/generate_inventory.py \
            --instance-ids "$TARGET_INSTANCE_IDS" \
            --region "$AWS_DEFAULT_REGION" \
            --output "$INVENTORY_FILE" \
            --aws-access-key-id "$CLIENT_ACCESS_KEY_ID" \
            --aws-secret-access-key "$CLIENT_SECRET_ACCESS_KEY" \
            --aws-session-token "$CLIENT_SESSION_TOKEN" \
            --s3-bucket "$S3_BUCKET_NAME" 2>&1; then
            return 0
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $GENERATE_INVENTORY_MAX_RETRIES ]; then
                echo "[entrypoint] WARNING: Inventory generation failed, retrying in ${retry_delay}s (attempt $retry_count/$GENERATE_INVENTORY_MAX_RETRIES)"
                sleep $retry_delay
                retry_delay=$((retry_delay * 2))
            else
                return 1
            fi
        fi
    done
}

if ! generate_inventory_with_retry; then
    echo "[entrypoint] ERROR: Failed to generate inventory after $GENERATE_INVENTORY_MAX_RETRIES attempts"
    upload_logs || true
    update_job_status "FAILED" "Failed to generate inventory from instance IDs"
    exit 1
fi

echo "[entrypoint] Inventory generated at $INVENTORY_FILE"

# 5. Build extra vars argument if provided
EXTRA_VARS_ARGS=()
if [ -n "${EXTRA_VARS:-}" ] && [ "$EXTRA_VARS" != "null" ] && [ "$EXTRA_VARS" != "{}" ]; then
    echo "$EXTRA_VARS" > /playbooks/extra_vars.json
    EXTRA_VARS_ARGS=("-e" "@/playbooks/extra_vars.json")
fi

# 6. Run Ansible playbook
echo "[entrypoint] Executing ansible-playbook..."
cd /playbooks

# Verify ansible-playbook is available before execution
if ! command -v ansible-playbook &>/dev/null; then
    echo "[entrypoint] ERROR: ansible-playbook command not found"
    update_job_status "FAILED" "ansible-playbook binary not available"
    upload_logs || true
    exit 1
fi

set +e
timeout "$PLAYBOOK_TIMEOUT" ansible-playbook main.yml "${EXTRA_VARS_ARGS[@]}" -v
EXIT_CODE=$?
set -e

# Handle timeout exit code (124)
if [ $EXIT_CODE -eq 124 ]; then
    echo "[entrypoint] ERROR: Playbook execution timed out after ${PLAYBOOK_TIMEOUT}s"
    stop_ec2_instances
    update_job_status "FAILED" "Playbook execution timeout after ${PLAYBOOK_TIMEOUT}s"
    upload_logs || true
    exit 1
fi

if [ $EXIT_CODE -eq 0 ]; then
    echo "[entrypoint] Ansible playbook completed successfully"
    stop_ec2_instances
    update_job_status "COMPLETED"
    upload_logs || true
    exit 0
else
    echo "[entrypoint] Ansible playbook failed with exit code: $EXIT_CODE"
    stop_ec2_instances
    update_job_status "FAILED" "Ansible playbook exited with code $EXIT_CODE"
    upload_logs || true
    exit $EXIT_CODE
fi
