#!/usr/bin/env bash
# Unit testing: System B's components one at a time.
# The tests are Go, in backend/adapter/tests/unit, because they use B's
# internal packages; this script runs them and writes tests/results/unit.json.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
run_go_suite "unit" "${REPO}/backend/adapter" "./tests/unit/" -v
