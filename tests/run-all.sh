#!/usr/bin/env bash
# Runs every test suite that can run today, in order, and writes each one's
# results to tests/results/ (or RESULTS_DIR). It keeps going past a failed
# suite and prints a table at the end; the exit status is 1 if any failed.
#
# The suites that need the running app (security-regression's live part and
# e2e-browser) check for it and say when they skipped; start it first with
# npm run dev for a full run.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/lib/common.sh"
date -u +%Y-%m-%dT%H:%M:%SZ >"${RESULTS}/started"
git -C "${REPO}" rev-parse HEAD >"${RESULTS}/revision"

suites=(platform-version unit connector-contract input-validation state-isolation performance security-regression aws-regression e2e-browser)
declare -A outcome
for s in "${suites[@]}"; do
  start=$(date +%s)
  bash "${HERE}/${s}/run.sh" 2>&1 | tee "${RESULTS}/${s}.console"
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
