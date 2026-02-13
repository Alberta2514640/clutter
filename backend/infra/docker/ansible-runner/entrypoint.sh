#!/bin/bash
set -e

# Validate Environment Variables
if [ -z "$JOB_ID" ]; then
    echo "ERROR: JOB_ID is not set"
    exit 1
fi
if [ -z "$PLAYBOOK_S3_KEY" ]; then
    echo "ERROR: PLAYBOOK_S3_KEY is not set"
    exit 1
fi
if [ -z "$S3_BUCKET_NAME" ]; then
    echo "ERROR: S3_BUCKET_NAME is not set"
    exit 1
fi
if [ -z "$PSQL_CONNECTION_STRING" ]; then
    echo "ERROR: PSQL_CONNECTION_STRING is not set"
    exit 1
fi

echo "Starting Ansible Runner for Job ID: $JOB_ID"

# Function to update job status in PostgreSQL
update_status() {
    local status=$1
    local message=$2
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    if [ -n "$message" ]; then
        psql "$PSQL_CONNECTION_STRING" -c "UPDATE jobs SET status = '$status', error_message = '$message', updated_at = '$timestamp' WHERE id = '$JOB_ID';"
    else
        psql "$PSQL_CONNECTION_STRING" -c "UPDATE jobs SET status = '$status', updated_at = '$timestamp' WHERE id = '$JOB_ID';"
    fi
}

# Update status to RUNNING
update_status "RUNNING" ""

# Download Playbook from S3
echo "Downloading playbook from s3://$S3_BUCKET_NAME/$PLAYBOOK_S3_KEY..."
aws s3 cp "s3://$S3_BUCKET_NAME/$PLAYBOOK_S3_KEY" playbook.yml

# Parse Extra Vars if provided
ANSIBLE_ARGS=""
if [ -n "$EXTRA_VARS" ]; then
    echo "$EXTRA_VARS" > extra_vars.json
    ANSIBLE_ARGS="-e @extra_vars.json"
fi

# Run Ansible Playbook
echo "Running Ansible Playbook..."
if ansible-playbook playbook.yml $ANSIBLE_ARGS; then
    echo "Ansible Playbook completed successfully."
    update_status "COMPLETED" ""
else
    echo "Ansible Playbook failed."
    update_status "FAILED" "Ansible execution failed"
    exit 1
fi
