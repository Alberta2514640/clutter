#!/usr/bin/env bash
# Performance testing: 20 Capability Check requests and 5 deployments through
# B API, timed, with the median and maximum. The test is Go, in
# backend/adapter/tests/performance; the numbers also go to
# tests/results/performance-numbers.json.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
export PERF_RESULTS="${RESULTS}/performance-numbers.json"
run_go_suite performance "${REPO}/backend/adapter" ./tests/performance/ -v
