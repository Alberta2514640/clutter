#!/usr/bin/env bash
# State isolation and security testing: simultaneous deployments and secret scans.
# The tests are Go, in backend/adapter/tests/isolation, because they use B's
# internal packages; this script runs them and writes tests/results/state-isolation.json.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
run_go_suite "state-isolation" "${REPO}/backend/adapter" "./tests/isolation/" -v -timeout 10m
