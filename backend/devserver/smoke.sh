#!/usr/bin/env bash
# Exercises all 31 API routes through the running devserver and prints a table.
# Start the devserver with run.sh first.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

[[ -s "${STATE_DIR}/pg_password" ]] || { echo "smoke.sh: run run.sh first" >&2; exit 1; }
load_secrets
curl -fs --max-time 2 "http://${API_ADDR}/dev/healthz" >/dev/null || { echo "smoke.sh: the gateway is not running; run run.sh first" >&2; exit 1; }

PSQL_CONNECTION_STRING="${PSQL_CONNECTION_STRING}" go -C "${DEVSERVER_DIR}" run ./cmd/smoke -api "http://${API_ADDR}"
