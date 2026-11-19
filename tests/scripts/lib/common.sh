#!/usr/bin/env bash
# Shared settings for every test suite's scripts (see tests/README.md).
# Source it; do not run it.

# REPO is the repository root; RESULTS is where suites write their results
# (tests/results unless RESULTS_DIR says otherwise).
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
RESULTS="${RESULTS_DIR:-${REPO}/tests/results}"
mkdir -p "${RESULTS}"
export REPO RESULTS

# run_go_suite NAME MODULE_DIR PACKAGE [go test flags...]
# Runs one Go package's tests with -json, keeps the raw output, and writes
# RESULTS/NAME.json through gotest_summary.py. Returns the suite's status.
run_go_suite() {
  local name="$1" module="$2" pkg="$3"
  shift 3
  local raw="${RESULTS}/${name}.jsonl"
  echo "== ${name}: go test ${pkg} (in ${module#"${REPO}/"})"
  set +e
  go -C "${module}" test -count=1 -json "$@" "${pkg}" >"${raw}" 2>"${RESULTS}/${name}.stderr"
  set -e
  python3 "${REPO}/tests/scripts/lib/gotest_summary.py" "${name}" "${raw}" "${RESULTS}/${name}.json"
}
