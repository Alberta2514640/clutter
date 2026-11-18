#!/usr/bin/env bash
# Connector contract testing: one shared suite against every connector.
# The tests are Go, in backend/adapter/tests/contract, because they use B's
# internal packages; this script runs them and writes tests/results/connector-contract.json.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
run_go_suite "connector-contract" "${REPO}/backend/adapter" "./tests/contract/" -v -timeout 10m
