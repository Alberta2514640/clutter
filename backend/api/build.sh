# This script builds each Go file in LAMBDA_DIRS into a deployable AWS Lambda binary and zip to be uploaded via Terraform
# Update the list when creating a new Lambda function

# Credit: ChatGPT

#!/usr/bin/env bash
set -euo pipefail

# Define a list of directories that contain main.go files
# (relative to this script's location)
LAMBDA_DIRS=(
  "diagram/create"
)

# Detect root directory of script (so you can run it from anywhere)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Cross-compile for AWS Lambda (Go custom runtime)
GOOS=linux
GOARCH=arm64

echo "🏗️  Building Go Lambda functions..."
for dir in "${LAMBDA_DIRS[@]}"; do
  SRC_DIR="${ROOT_DIR}/${dir}"
  DEPLOY_DIR="${SRC_DIR}/deploy"

  echo "📁 Building Lambda in: ${dir}"

  # Create deploy directory if it doesn’t exist
  mkdir -p "${DEPLOY_DIR}"

  # Build the binary as 'bootstrap'
  echo "   → Compiling Go source..."
  GOOS=$GOOS GOARCH=$GOARCH go build -o "${DEPLOY_DIR}/bootstrap" "${SRC_DIR}/main.go"

# Zip the binary (Windows PowerShell compatible)
if command -v zip &> /dev/null; then
  cd "${DEPLOY_DIR}"
  zip -q -r bootstrap.zip bootstrap
  cd - > /dev/null
else
  # Convert /c/... path to C:\... for PowerShell
  WIN_PATH=$(echo "${DEPLOY_DIR}" | sed 's|^/c|C:|' | sed 's|/|\\|g')
  echo "   → Zipping with PowerShell at: ${WIN_PATH}\\bootstrap.zip"
  powershell.exe -Command "Compress-Archive -Path '${WIN_PATH}\\bootstrap' -DestinationPath '${WIN_PATH}\\bootstrap.zip' -Force"
fi

  echo "✅ Built ${dir}/deploy/bootstrap.zip"
done

echo "🎉 All Lambdas built successfully!"
