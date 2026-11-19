#!/usr/bin/env bash
# Input and capability validation testing: the approved set of invalid inputs.
# Runs the Go tests in backend/adapter/tests/validation and writes
# test-suites/results/input-validation.json. See backend/adapter/tests/README.md.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/test-suites/scripts/lib/common.sh"
run_go_suite input-validation "${REPO}/backend/adapter" ./tests/validation/ -v
