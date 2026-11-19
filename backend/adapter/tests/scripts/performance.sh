#!/usr/bin/env bash
# Performance testing: 20 Capability Check requests and 5 deployments through
# B API, timed, with the median and maximum. Runs the Go test in
# backend/adapter/tests/performance; the numbers also go to
# tests/results/performance-numbers.json. See backend/adapter/tests/README.md.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tests/scripts/lib/common.sh"
export PERF_RESULTS="${RESULTS}/performance-numbers.json"
run_go_suite performance "${REPO}/backend/adapter" ./tests/performance/ -v
