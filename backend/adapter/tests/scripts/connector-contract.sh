#!/usr/bin/env bash
# Connector contract testing: one shared suite against every connector.
# Runs the Go tests in backend/adapter/tests/contract and writes
# test-suites/results/connector-contract.json. See backend/adapter/tests/README.md.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/test-suites/scripts/lib/common.sh"
run_go_suite connector-contract "${REPO}/backend/adapter" ./tests/contract/ -v -timeout 10m
