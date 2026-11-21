#!/usr/bin/env bash
# Runs every test suite that can run today, in order, and writes each one's
# results to test-suites/results/ (or RESULTS_DIR). It keeps going past a failed
# suite and prints a table at the end; the exit status is 1 if any failed.
#
# The suites that need the running app (security-regression's live part and
# e2e-browser) check for it and say when they skipped; start it first with
# npm run dev for a full run.
set -uo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/test-suites/scripts/lib/common.sh"
date -u +%Y-%m-%dT%H:%M:%SZ >"${RESULTS}/started"
git -C "${REPO}" rev-parse HEAD >"${RESULTS}/revision"

# Each suite's script sits beside the code it tests (test-suites/README.md).
# Plain arrays and a case, not declare -A: macOS's own bash 3.2 runs this.
script_of() {
  case "$1" in
    platform-version) echo test-suites/scripts/platform-version/run.sh ;;
    unit) echo backend/adapter/tests/scripts/unit.sh ;;
    connector-contract) echo backend/adapter/tests/scripts/connector-contract.sh ;;
    input-validation) echo backend/adapter/tests/scripts/input-validation.sh ;;
    state-isolation) echo backend/adapter/tests/scripts/state-isolation.sh ;;
    performance) echo backend/adapter/tests/scripts/performance.sh ;;
    security-regression) echo backend/api/tests/scripts/security-regression/run.sh ;;
    aws-regression) echo backend/api/tests/scripts/aws-regression/run.sh ;;
    e2e-browser) echo frontend/clutter/tests/scripts/e2e-browser.sh ;;
  esac
}
suites=(platform-version unit connector-contract input-validation state-isolation performance security-regression aws-regression e2e-browser)
outcomes=()
for s in "${suites[@]}"; do
  start=$(date +%s)
  bash "${REPO}/$(script_of "${s}")" 2>&1 | tee "${RESULTS}/${s}.console"
  code=${PIPESTATUS[0]}
  case "${code}" in
    0) outcomes+=("pass") ;;
    2) outcomes+=("skipped") ;;
    *) outcomes+=("FAIL") ;;
  esac
  echo "$(( $(date +%s) - start ))" >"${RESULTS}/${s}.seconds"
done

echo
printf '%-22s %s\n' SUITE RESULT
failed=0
for i in "${!suites[@]}"; do
  printf '%-22s %s\n' "${suites[$i]}" "${outcomes[$i]}"
  [[ "${outcomes[$i]}" == FAIL ]] && failed=1
done
date -u +%Y-%m-%dT%H:%M:%SZ >"${RESULTS}/finished"
exit "${failed}"
