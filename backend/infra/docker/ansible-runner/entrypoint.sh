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
#
# Optional:
#   EXTRA_VARS          - JSON string of extra variables for the playbook

LOG_FILE="/var/log/ansible/ansible.log"
INVENTORY_FILE="/playbooks/inventory.yml"

# ---- Helper: Update job status in PostgreSQL (parameterized, no SQL injection) ----
update_job_status() {
    local status="$1"
    local error_message="${2:-}"

    python3 - <<'PYEOF' "$PSQL_CONNECTION_STRING" "$JOB_ID" "$status" "$error_message"
import sys
import psycopg2
from datetime import datetime, timezone

conn_string, job_id, status, error_message = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
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
    cur.close()
    conn.close()
except Exception as e:
    print(f"[update_job_status] WARNING: Failed to update job status: {e}", file=sys.stderr)
PYEOF
}

# ---- Helper: Upload logs to S3 ----
upload_logs() {
    if [ -f "$LOG_FILE" ]; then
        aws s3 cp "$LOG_FILE" "s3://${S3_BUCKET_NAME}/logs/${JOB_ID}/ansible.log" \
            --region "$AWS_DEFAULT_REGION" 2>/dev/null || true
    fi
}

# ---- Trap: Handle signals for graceful shutdown ----
cleanup() {
    echo "[entrypoint] Received shutdown signal. Uploading logs and marking as FAILED."
    upload_logs
    update_job_status "FAILED" "Task was terminated by signal"
    exit 1
}
trap cleanup SIGTERM SIGINT

# ---- Validate required env vars ----
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-${AWS_REGION:-}}"
for var in JOB_ID PLAYBOOK_S3_KEY TARGET_INSTANCE_IDS S3_BUCKET_NAME PSQL_CONNECTION_STRING AWS_DEFAULT_REGION; do
    if [ -z "${!var:-}" ]; then
        echo "[entrypoint] ERROR: Missing required environment variable: $var"
        update_job_status "FAILED" "Missing environment variable: $var"
        exit 1
    fi
done

echo "[entrypoint] Starting Ansible job: $JOB_ID"
echo "[entrypoint] Playbook: s3://${S3_BUCKET_NAME}/${PLAYBOOK_S3_KEY}"
echo "[entrypoint] Targets: $TARGET_INSTANCE_IDS"
echo "[entrypoint] Connection: SSM Session Manager"

# 1. Mark job as RUNNING
update_job_status "RUNNING"

# 2. Download playbook from S3
echo "[entrypoint] Downloading playbook from S3..."
if ! aws s3 cp "s3://${S3_BUCKET_NAME}/${PLAYBOOK_S3_KEY}" /playbooks/main.yml --region "$AWS_DEFAULT_REGION"; then
    echo "[entrypoint] ERROR: Failed to download playbook"
    update_job_status "FAILED" "Failed to download playbook from S3"
    exit 1
fi

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

[privilege_escalation]
become = True
become_method = sudo
ANSIBLECFG

export ANSIBLE_CONFIG=/playbooks/ansible.cfg

# 4. Generate dynamic inventory from instance IDs (uses SSM connection)
echo "[entrypoint] Generating inventory..."
if ! python3 /usr/local/bin/generate_inventory.py \
    --instance-ids "$TARGET_INSTANCE_IDS" \
    --region "$AWS_DEFAULT_REGION" \
    --s3-bucket "$S3_BUCKET_NAME" \
    --output "$INVENTORY_FILE"; then
    echo "[entrypoint] ERROR: Failed to generate inventory"
    update_job_status "FAILED" "Failed to generate inventory from instance IDs"
    upload_logs
    exit 1
fi

echo "[entrypoint] Generated inventory:"
cat "$INVENTORY_FILE"

# 5. Build extra vars argument if provided
EXTRA_VARS_ARGS=""
if [ -n "${EXTRA_VARS:-}" ] && [ "$EXTRA_VARS" != "null" ] && [ "$EXTRA_VARS" != "{}" ]; then
    echo "$EXTRA_VARS" > /playbooks/extra_vars.json
    EXTRA_VARS_ARGS="-e @/playbooks/extra_vars.json"
fi

# 6. Run Ansible playbook
echo "[entrypoint] Executing ansible-playbook..."
cd /playbooks
set +e
ansible-playbook main.yml $EXTRA_VARS_ARGS -v
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 0 ]; then
    echo "[entrypoint] Ansible playbook completed successfully"
    upload_logs
    update_job_status "COMPLETED"
    exit 0
else
    echo "[entrypoint] Ansible playbook failed with exit code: $EXIT_CODE"
    upload_logs
    update_job_status "FAILED" "Ansible playbook exited with code $EXIT_CODE"
    exit $EXIT_CODE
fi
