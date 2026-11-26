#!/usr/bin/env bash
# State isolation and security testing: simultaneous deployments and secret scans.
# Runs the Go tests in backend/adapter/tests/isolation and writes
# test-suites/results/state-isolation.json. See backend/adapter/tests/README.md.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/test-suites/scripts/lib/common.sh"
run_go_suite state-isolation "${REPO}/backend/adapter" ./tests/isolation/ -v -timeout 10m
