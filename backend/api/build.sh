#!/usr/bin/env bash
set -euo pipefail

# Define a list of directories that contain main.go files
LAMBDA_DIRS=(
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

for dir in "${LAMBDA_DIRS[@]}"; do
  SRC_DIR="${ROOT_DIR}/${dir}"
  DEPLOY_DIR="${SRC_DIR}/deploy"
  MAIN_GO="${SRC_DIR}/main.go"
  BINARY="${DEPLOY_DIR}/bootstrap"
  ZIP_FILE="${DEPLOY_DIR}/bootstrap.zip"

  echo "📁 Checking Lambda in: ${dir}"

  mkdir -p "${DEPLOY_DIR}"

  # Only rebuild if main.go is newer than the binary or if binary doesn't exist
  if [[ ! -f "$BINARY" ]] || [[ "$MAIN_GO" -nt "$BINARY" ]]; then
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