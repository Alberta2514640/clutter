#!/usr/bin/env bash
# Unit testing: System B's components one at a time.
# Runs the Go tests in backend/adapter/tests/unit and writes
# tests/results/unit.json. See backend/adapter/tests/README.md.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tests/scripts/lib/common.sh"
run_go_suite unit "${REPO}/backend/adapter" ./tests/unit/ -v
