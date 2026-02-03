#!/usr/bin/env bash
set -euo pipefail

TOTAL_START_TIME=$(date +%s)

echo "Starting Terraform deploy."

# ENVIRONMENT VARIABLES (Required)
# ===============================
# AWS_REGION
# TERRAFORM_DIRECTORY
# CLIENT_ROLE_ARN
# ASSUME_ROLE_EXTERNAL_ID

# -------------------------------
# Log files
# -------------------------------
INIT_LOG="/tmp/terraform-init.log"
APPLY_LOG="/tmp/terraform-apply.log"

# -------------------------------
# Error handler
# -------------------------------
on_error() {
  exit_code=$?

  echo
  echo "Terraform deploy failed (exit code: $exit_code)"
  echo "Failed command: ${BASH_COMMAND}"
  echo

  if [[ -f "$INIT_LOG" ]]; then
    echo "Terraform init output:"
    echo "----------------------"
    cat "$INIT_LOG"
    echo "----------------------"
  fi

  if [[ -f "$APPLY_LOG" ]]; then
    echo "Terraform apply output:"
    echo "-----------------------"
    cat "$APPLY_LOG"
    echo "-----------------------"
  fi

  exit "$exit_code"
}

trap on_error ERR

# -------------------------------
# Fetch Terraform code
# -------------------------------
echo "Downloading Terraform from S3."
aws s3 sync "s3://clutter-us-west-2-446fe866/$TERRAFORM_DIRECTORY" /app

# -------------------------------
# Assume customer role
# -------------------------------
echo "Assuming customer role."

CREDS=$(aws sts assume-role \
  --role-arn "$CLIENT_ROLE_ARN" \
  --role-session-name clutter-terraform-deployer-session \
  --external-id "$ASSUME_ROLE_EXTERNAL_ID")

export AWS_ACCESS_KEY_ID=$(echo "$CREDS" | jq -r '.Credentials.AccessKeyId')
export AWS_SECRET_ACCESS_KEY=$(echo "$CREDS" | jq -r '.Credentials.SecretAccessKey')
export AWS_SESSION_TOKEN=$(echo "$CREDS" | jq -r '.Credentials.SessionToken')

INIT_LOG="/tmp/terraform-init.log"
APPLY_LOG="/tmp/terraform-apply.log"

# -------------------------------
# Terraform Init
# -------------------------------
echo "Initializing Terraform."
INIT_START_TIME=$(date +%s)

terraform init >"$INIT_LOG" 2>&1

INIT_END_TIME=$(date +%s)
INIT_DURATION=$((INIT_END_TIME - INIT_START_TIME))
INIT_MIN=$((INIT_DURATION / 60))
INIT_SEC=$((INIT_DURATION % 60))

echo "Terraform init time: ${INIT_MIN}m ${INIT_SEC}s"
echo "Terraform init output:"
echo "----------------------"
cat "$INIT_LOG"
echo "----------------------"

# -------------------------------
# Terraform Apply
# -------------------------------
echo "Applying Terraform."
APPLY_START_TIME=$(date +%s)

terraform apply -auto-approve >"$APPLY_LOG" 2>&1

APPLY_END_TIME=$(date +%s)
APPLY_DURATION=$((APPLY_END_TIME - APPLY_START_TIME))
APPLY_MIN=$((APPLY_DURATION / 60))
APPLY_SEC=$((APPLY_DURATION % 60))

echo "Terraform apply time: ${APPLY_MIN}m ${APPLY_SEC}s"
echo "Terraform apply output:"
echo "-----------------------"
cat "$APPLY_LOG"
echo "-----------------------"

# -------------------------------
# Totals
# -------------------------------
TOTAL_END_TIME=$(date +%s)
TOTAL_DURATION=$((TOTAL_END_TIME - TOTAL_START_TIME))
TOTAL_MIN=$((TOTAL_DURATION / 60))
TOTAL_SEC=$((TOTAL_DURATION % 60))

echo "Terraform deploy complete."
echo "Total time taken: ${TOTAL_MIN}m ${TOTAL_SEC}s"
