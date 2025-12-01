#!/usr/bin/env bash
set -euo pipefail

# Define a list of directories that contain main.go files
LAMBDA_DIRS=(
  "log-in"
  "authorizer"
  "organization/create"
  "organization/delete"
  "organization/get"
  "organization/overview"
  "organization/update"
  "project/create"
  "project/delete"
  "project/get"
  "project/update"
  "diagram/create"
  "diagram/get"
  "diagram/update"
  "diagram/delete"
)

# Detect root directory of script
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Cross-compile for AWS Lambda (Go custom runtime)
GOOS=linux
GOARCH=arm64

echo "🏗️  Building Go Lambda functions..."

# Compute latest modification time in generic folder
GENERIC_DIR="${ROOT_DIR}/generic"
GENERIC_MOD_TIME=$(find "$GENERIC_DIR" -type f -printf "%T@\n" | sort -n | tail -1 | awk '{print int($1)}' || echo 0)

for dir in "${LAMBDA_DIRS[@]}"; do
  SRC_DIR="${ROOT_DIR}/${dir}"
  DEPLOY_DIR="${SRC_DIR}/deploy"
  MAIN_GO="${SRC_DIR}/main.go"
  BINARY="${DEPLOY_DIR}/bootstrap"
  ZIP_FILE="${DEPLOY_DIR}/bootstrap.zip"

  echo "📁 Checking Lambda in: ${dir}"

  mkdir -p "${DEPLOY_DIR}"

  # Get modification time of Lambda's main.go
  MAIN_MOD_TIME=$(stat -c %Y "$MAIN_GO" 2>/dev/null || echo 0)
  BINARY_MOD_TIME=$(stat -c %Y "$BINARY" 2>/dev/null || echo 0)


  # Rebuild if main.go is newer than binary, or binary missing, or generic changed
  if [[ ! -f "$BINARY" ]] || [[ "$MAIN_MOD_TIME" -gt "$BINARY_MOD_TIME" ]] || [[ "$GENERIC_MOD_TIME" -gt "$BINARY_MOD_TIME" ]]; then
    echo "   → Compiling Go source..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$BINARY" "$MAIN_GO"

    # Zip the binary
    if command -v zip &> /dev/null; then
      cd "$DEPLOY_DIR"
      zip -q -r bootstrap.zip bootstrap
      cd - > /dev/null
    else
      WIN_PATH=$(echo "$DEPLOY_DIR" | sed 's|^/c|C:|' | sed 's|/|\\|g')
      echo "   → Zipping with PowerShell at: ${WIN_PATH}\\bootstrap.zip"
      powershell.exe -Command "Compress-Archive -Path '${WIN_PATH}\\bootstrap' -DestinationPath '${WIN_PATH}\\bootstrap.zip' -Force"
    fi

    echo "✅ Built ${dir}/deploy/bootstrap.zip"
  else
    echo "   → No changes detected, skipping build."
  fi
done

echo "🎉 All Lambdas processed successfully!"