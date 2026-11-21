#!/usr/bin/env bash
# Exercises all 32 API routes through the running devserver and prints a table,
# then checks B API's main paths in a section of its own.
# Start the backend first with dev.sh --backend-only (or dev.sh).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

case "$*" in
  "") ;;
  -h | --help)
    cat <<'EOF'
usage: npm run smoke

Calls all 32 of A's API routes through the running backend and prints a table,
then checks B API's main paths. Any FAIL makes it exit with 1.
Start the backend first: npm run dev (or backend/devserver/dev.sh --backend-only).
EOF
    exit 0
    ;;
  *) echo "usage: npm run smoke" >&2; exit 2 ;;
esac

[[ -s "${STATE_DIR}/pg_password" ]] || { echo "smoke.sh: start the backend first: dev.sh --backend-only" >&2; exit 1; }
load_secrets
curl -fs --max-time 2 "http://${API_ADDR}/dev/healthz" >/dev/null || { echo "smoke.sh: the gateway is not running; start it with dev.sh --backend-only" >&2; exit 1; }
adapter_healthy || { echo "smoke.sh: B API is not running on ${ADAPTER_ADDR}; start the backend with dev.sh --backend-only" >&2; exit 1; }

PSQL_CONNECTION_STRING="${PSQL_CONNECTION_STRING}" JWT_SECRET="${JWT_SECRET}" go -C "${DEVSERVER_DIR}" run ./cmd/smoke -api "http://${API_ADDR}" -adapter-api "http://${ADAPTER_ADDR}" -s3-dir "${STATE_DIR}/s3"
