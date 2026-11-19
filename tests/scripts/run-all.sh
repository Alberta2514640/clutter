#!/usr/bin/env bash
# Runs every test suite that can run today, in order, and writes each one's
# results to tests/results/ (or RESULTS_DIR). It keeps going past a failed
# suite and prints a table at the end; the exit status is 1 if any failed.
#
# The suites that need the running app (security-regression's live part and
# e2e-browser) check for it and say when they skipped; start it first with
# npm run dev for a full run.
set -uo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tests/scripts/lib/common.sh"
date -u +%Y-%m-%dT%H:%M:%SZ >"${RESULTS}/started"
git -C "${REPO}" rev-parse HEAD >"${RESULTS}/revision"

# Each suite's script sits beside the code it tests (tests/README.md).
declare -A script=(
  [platform-version]=tests/scripts/platform-version/run.sh
  [unit]=backend/adapter/tests/scripts/unit.sh
  [connector-contract]=backend/adapter/tests/scripts/connector-contract.sh
  [input-validation]=backend/adapter/tests/scripts/input-validation.sh
  [state-isolation]=backend/adapter/tests/scripts/state-isolation.sh
  [performance]=backend/adapter/tests/scripts/performance.sh
  [security-regression]=backend/api/tests/scripts/security-regression/run.sh
  [aws-regression]=backend/api/tests/scripts/aws-regression/run.sh
  [e2e-browser]=frontend/clutter/tests/scripts/e2e-browser.sh
)
suites=(platform-version unit connector-contract input-validation state-isolation performance security-regression aws-regression e2e-browser)
declare -A outcome
for s in "${suites[@]}"; do
  start=$(date +%s)
  bash "${REPO}/${script[${s}]}" 2>&1 | tee "${RESULTS}/${s}.console"
  code=${PIPESTATUS[0]}
  case "${code}" in
    0) outcome[${s}]="pass" ;;
    2) outcome[${s}]="skipped" ;;
    *) outcome[${s}]="FAIL" ;;
  esac
  echo "$(( $(date +%s) - start ))" >"${RESULTS}/${s}.seconds"
done

echo
printf '%-22s %s\n' SUITE RESULT
failed=0
for s in "${suites[@]}"; do
  printf '%-22s %s\n' "${s}" "${outcome[${s}]}"
  [[ "${outcome[${s}]}" == FAIL ]] && failed=1
done
date -u +%Y-%m-%dT%H:%M:%SZ >"${RESULTS}/finished"
exit "${failed}"
