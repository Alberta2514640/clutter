#!/bin/bash
set -euo pipefail

# =============================================================================
# Terraform Runner Entrypoint Script
# =============================================================================
# This script runs Terraform deployments in a Fargate container.
# It downloads the Terraform workspace from S3, runs init/plan/apply,
# and uploads the logs back to S3 for persistence.
#
# Environment variables expected:
# - PROJECT_ID: UUID of the project
# - RUN_ID: UUID of this deployment run
# - USER_ID: UUID of the user triggering deployment
# - S3_BUCKET: S3 bucket name for terraform workspaces and logs
# - AWS_REGION: AWS region (default: us-east-1)
# - TERRAFORM_ACTION: Action to perform (plan, apply, destroy) - default: apply
# =============================================================================

# Exit codes
EXIT_SUCCESS=0
EXIT_DOWNLOAD_FAILED=1
EXIT_INIT_FAILED=2
EXIT_PLAN_FAILED=3
EXIT_APPLY_FAILED=4
EXIT_UPLOAD_FAILED=5
EXIT_INVALID_ACTION=6

# Configuration
AWS_REGION=${AWS_REGION:-us-east-1}
TERRAFORM_ACTION=${TERRAFORM_ACTION:-apply}
LOG_FILE="/tmp/terraform.log"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
EXIT_CODE=$EXIT_SUCCESS

# Function to log messages
log() {
    echo "[$(date -u +"%Y-%m-%d %H:%M:%S")] $1" | tee -a "$LOG_FILE"
}

# Function to upload logs to S3 (called on exit)
upload_logs() {
    local final_exit_code=$1
    log "Uploading logs to S3..."
    
    # Add exit code metadata to log
    echo "" >> "$LOG_FILE"
    echo "========================================" >> "$LOG_FILE"
    echo "Exit Code: $final_exit_code" >> "$LOG_FILE"
    echo "========================================" >> "$LOG_FILE"
    
    if aws s3 cp "$LOG_FILE" "s3://${S3_BUCKET}/${LOG_PATH}" \
        --metadata "exit-code=${final_exit_code},run-id=${RUN_ID},project-id=${PROJECT_ID}"; then
        log "✓ Logs uploaded successfully"
    else
        log "✗ Failed to upload logs to S3"
    fi
    
    exit $final_exit_code
}

# Validate required environment variables
validate_environment() {
    local missing_vars=()
    
    [[ -z "${PROJECT_ID:-}" ]] && missing_vars+=("PROJECT_ID")
    [[ -z "${RUN_ID:-}" ]] && missing_vars+=("RUN_ID")
    [[ -z "${S3_BUCKET:-}" ]] && missing_vars+=("S3_BUCKET")
    [[ -z "${USER_ID:-}" ]] && missing_vars+=("USER_ID")
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log "✗ Missing required environment variables: ${missing_vars[*]}"
        # If S3_BUCKET is missing, we cannot upload logs - just exit
        if [[ " ${missing_vars[*]} " =~ " S3_BUCKET " ]]; then
            echo "ERROR: Cannot upload logs - S3_BUCKET is not set" >&2
            exit 1
        fi
        upload_logs $EXIT_DOWNLOAD_FAILED
    fi
}

# Initialize log file
> "$LOG_FILE"

# Validate environment before proceeding (must be called before using PROJECT_ID, RUN_ID, etc.)
validate_environment

# Set paths after validation (these require PROJECT_ID and RUN_ID to be set)
TERRAFORM_WORKSPACE_PATH="terraform-workspaces/${PROJECT_ID}"
LOG_PATH="logs/${RUN_ID}.log"

echo "========================================" | tee -a "$LOG_FILE"
echo "Terraform Runner - Starting Deployment" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
echo "Run ID: ${RUN_ID}" | tee -a "$LOG_FILE"
echo "Project ID: ${PROJECT_ID}" | tee -a "$LOG_FILE"
echo "User ID: ${USER_ID}" | tee -a "$LOG_FILE"
echo "Action: ${TERRAFORM_ACTION}" | tee -a "$LOG_FILE"
echo "Timestamp: ${TIMESTAMP}" | tee -a "$LOG_FILE"
echo "Terraform Version: $(terraform --version -json | jq -r '.terraform_version')" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

# Step 1: Download terraform workspace from S3
log "Downloading Terraform workspace from S3..."
if aws s3 cp "s3://${S3_BUCKET}/${TERRAFORM_WORKSPACE_PATH}/" /terraform --recursive 2>&1 | tee -a "$LOG_FILE"; then
    if ! cd /terraform; then
        log "✗ Failed to change directory to /terraform"
        upload_logs $EXIT_DOWNLOAD_FAILED
    fi
    log "✓ Workspace downloaded successfully"
else
    log "✗ Failed to download workspace"
    upload_logs $EXIT_DOWNLOAD_FAILED
fi

echo "" | tee -a "$LOG_FILE"

# Step 2: Initialize Terraform with S3 backend
log "Initializing Terraform..."
if terraform init \
    -backend-config="bucket=${S3_BUCKET}" \
    -backend-config="key=terraform-states/${PROJECT_ID}/terraform.tfstate" \
    -backend-config="region=${AWS_REGION}" \
    -no-color 2>&1 | tee -a "$LOG_FILE"; then
    log "✓ Terraform initialized successfully"
else
    log "✗ Terraform initialization failed"
    upload_logs $EXIT_INIT_FAILED
fi

echo "" | tee -a "$LOG_FILE"

# Step 3: Run terraform plan
log "Running Terraform plan..."
if terraform plan -no-color 2>&1 | tee -a "$LOG_FILE"; then
    log "✓ Terraform plan completed"
else
    log "✗ Terraform plan failed"
    upload_logs $EXIT_PLAN_FAILED
fi

echo "" | tee -a "$LOG_FILE"

# Step 4: Run terraform apply (or destroy based on TERRAFORM_ACTION)
case "$TERRAFORM_ACTION" in
    plan)
        log "Terraform action is 'plan' - skipping apply"
        ;;
    destroy)
        log "Running Terraform destroy..."
        if terraform destroy -auto-approve -no-color 2>&1 | tee -a "$LOG_FILE"; then
            log "✓ Destroy complete! Resources removed successfully"
        else
            log "✗ Terraform destroy failed"
            upload_logs $EXIT_APPLY_FAILED
        fi
        ;;
    apply)
        log "Running Terraform apply..."
        if terraform apply -auto-approve -no-color 2>&1 | tee -a "$LOG_FILE"; then
            log "✓ Apply complete! Resources deployed successfully"
        else
            log "✗ Terraform apply failed"
            upload_logs $EXIT_APPLY_FAILED
        fi
        ;;
    *)
        log "✗ Unknown TERRAFORM_ACTION: '$TERRAFORM_ACTION'. Valid values are: plan, apply, destroy"
        upload_logs $EXIT_INVALID_ACTION
        ;;
esac

echo "" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
echo "Deployment Completed Successfully" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"

# Step 5: Upload logs to S3
upload_logs $EXIT_SUCCESS
