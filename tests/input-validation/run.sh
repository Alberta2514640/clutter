#!/usr/bin/env bash
# Input and capability validation testing: the approved set of invalid inputs.
# The tests are Go, in backend/adapter/tests/validation, because they use B's
# internal packages; this script runs them and writes tests/results/input-validation.json.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
run_go_suite "input-validation" "${REPO}/backend/adapter" "./tests/validation/" -v
