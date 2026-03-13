#!/usr/bin/env bash
# =============================================================================
# Clutter Backend — Full Deployment Script
# Deploys infrastructure via Terraform, then builds and pushes the
# Ansible Runner Docker image to ECR.
#
# Prerequisites:
#   - AWS credentials configured for the clutter-test IAM user (us-west-2)
#   - terraform >= 1.0 on PATH
#   - docker on PATH
#   - aws CLI on PATH
#
# Usage:
#   cd /Users/syriljacob/Documents/CapstoneFinalProject/clutter/backend/infra
#   chmod +x deploy.sh
#   ./deploy.sh
# =============================================================================
set -euo pipefail

INFRA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKER_DIR="${INFRA_DIR}/docker/ansible-runner"
AWS_REGION="us-west-2"

log()  { echo "[deploy] $*"; }
fail() { echo "[deploy] ERROR: $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# STEP 0 — Kill any stuck Terraform provider processes
# ---------------------------------------------------------------------------
log "Step 0: Killing any stuck terraform provider processes..."
pkill -9 -f "terraform-provider-aws" 2>/dev/null && log "  Killed stuck provider process." || log "  No stuck provider process found."

# ---------------------------------------------------------------------------
# STEP 1 — Verify tool prerequisites
# ---------------------------------------------------------------------------
log "Step 1: Checking prerequisites..."
for tool in terraform aws docker; do
  command -v "$tool" >/dev/null 2>&1 || fail "$tool not found on PATH"
  log "  $tool: $(command -v "$tool")"
done
terraform version | head -1
aws --version
docker version --format "Docker {{.Client.Version}}"

# ---------------------------------------------------------------------------
# STEP 2 — Verify all Lambda ZIP files are present
# ---------------------------------------------------------------------------
log "Step 2: Verifying Lambda ZIPs..."
API_DIR="${INFRA_DIR}/../api"
LAMBDAS=(
  "log-in"
  "authorizer"
  "cloudformation/stack-url-generator"
  "organization/create"
  "organization/get"
  "organization/update"
  "organization/delete"
  "project/create"
  "project/get"
  "project/update"
  "project/delete"
  "diagram/create"
  "diagram/get"
  "diagram/update"
  "diagram/delete"
  "user-information/get"
  "terraform-engine/create"
  "ansible/create-playbook-upload-url"
  "ansible/get-job"
  "ansible/list-jobs"
  "ansible/run-task"
  "ansible/submit-job"
)
missing_zips=0
for lambda in "${LAMBDAS[@]}"; do
  zip_path="${API_DIR}/${lambda}/deploy/bootstrap.zip"
  if [ -f "$zip_path" ]; then
    log "  OK: ${lambda}"
  else
    log "  MISSING: ${zip_path}"
    missing_zips=$((missing_zips + 1))
  fi
done
if [ "$missing_zips" -gt 0 ]; then
  log ""
  log "  Some Lambda ZIPs are missing. Build them first:"
  log "    cd ${API_DIR} && ./build.sh"
  fail "Missing $missing_zips Lambda ZIP(s). Cannot proceed."
fi

# ---------------------------------------------------------------------------
# STEP 3 — Verify AWS credentials
# ---------------------------------------------------------------------------
log "Step 3: Verifying AWS credentials..."
CALLER=$(aws sts get-caller-identity --region "$AWS_REGION" --output json 2>&1) || fail "AWS credentials not valid or not configured.\nOutput: $CALLER"
ACCOUNT_ID=$(echo "$CALLER" | python3 -c "import sys,json; print(json.load(sys.stdin)['Account'])")
ARN=$(echo "$CALLER" | python3 -c "import sys,json; print(json.load(sys.stdin)['Arn'])")
log "  Account: $ACCOUNT_ID"
log "  Identity: $ARN"

# Warn if S3 backend bucket might not be reachable
log "  Checking S3 backend bucket access..."
aws s3 ls s3://tfstate-us-west-2-fc56b463 --region "$AWS_REGION" >/dev/null 2>&1 \
  && log "  S3 backend bucket accessible." \
  || log "  WARNING: Cannot access s3://tfstate-us-west-2-fc56b463. Terraform init may fail."

# ---------------------------------------------------------------------------
# STEP 4 — Terraform init (refresh backend)
# ---------------------------------------------------------------------------
log "Step 4: Running terraform init..."
cd "$INFRA_DIR"
terraform init -reconfigure -input=false 2>&1 | tail -5

# ---------------------------------------------------------------------------
# STEP 5 — Terraform plan
# ---------------------------------------------------------------------------
log "Step 5: Running terraform plan..."
terraform plan -var-file=terraform.tfvars -input=false -out=tfplan 2>&1
log "  Plan complete. Review the output above."

# ---------------------------------------------------------------------------
# STEP 6 — Terraform apply
# ---------------------------------------------------------------------------
log "Step 6: Applying terraform plan..."
terraform apply -input=false tfplan 2>&1
log "  Apply complete."
rm -f tfplan

# ---------------------------------------------------------------------------
# STEP 7 — Capture terraform outputs
# ---------------------------------------------------------------------------
log "Step 7: Capturing terraform outputs..."
TF_OUTPUT=$(terraform output -json 2>&1)
API_BASE_URL=$(echo "$TF_OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['api_base_url']['value'])")
CLUTTER_BUCKET=$(echo "$TF_OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['clutter_bucket_name']['value'])")
ECR_REPO_URL=$(terraform output -raw -no-color 2>/dev/null || true)

log "  API Base URL:      $API_BASE_URL"
log "  Clutter S3 Bucket: $CLUTTER_BUCKET"

# Get ECR URL from the fargate module output (not in root outputs.tf but accessible via state)
ECR_REPO_URL=$(terraform state show "module.fargate.aws_ecr_repository.ansible_runner" 2>/dev/null \
  | grep "repository_url" | awk '{print $3}' | tr -d '"' || true)
if [ -z "$ECR_REPO_URL" ]; then
  ECR_REPO_URL="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/clutter-ansible-runner"
  log "  ECR URL (derived): $ECR_REPO_URL"
else
  log "  ECR URL (from state): $ECR_REPO_URL"
fi

# ---------------------------------------------------------------------------
# STEP 8 — Build and push Ansible Runner Docker image
# ---------------------------------------------------------------------------
log "Step 8: Building and pushing Ansible Runner Docker image..."
cd "$DOCKER_DIR"

log "  Authenticating with ECR..."
aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

log "  Building image (linux/amd64)..."
docker build \
  --platform linux/amd64 \
  --provenance false \
  -t "clutter-ansible-runner:latest" \
  .

log "  Tagging image..."
docker tag "clutter-ansible-runner:latest" "${ECR_REPO_URL}:latest"

log "  Pushing to ECR..."
docker push "${ECR_REPO_URL}:latest"
log "  Pushed: ${ECR_REPO_URL}:latest"

# ---------------------------------------------------------------------------
# STEP 9 — Write .deployment-outputs for the e2e test script
# ---------------------------------------------------------------------------
cat > "${INFRA_DIR}/.deployment-outputs" <<EOF
API_BASE_URL=${API_BASE_URL}
CLUTTER_BUCKET=${CLUTTER_BUCKET}
ECR_REPO_URL=${ECR_REPO_URL}
AWS_REGION=${AWS_REGION}
EOF
log "  Outputs saved to ${INFRA_DIR}/.deployment-outputs"

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
log ""
log "Deployment complete."
log ""
log "  API Base URL:      ${API_BASE_URL}"
log "  Clutter S3 Bucket: ${CLUTTER_BUCKET}"
log "  ECR Image:         ${ECR_REPO_URL}:latest"
log ""
log "Run the end-to-end test:"
log "  cd ${INFRA_DIR} && ./e2e-test.sh"
