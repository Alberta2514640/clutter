#!/usr/bin/env bash
# AWS regression testing: run the agreed AWS baseline suite (every Go test of
# A, backend/api, which holds the AWS generator, its golden test, the
# runners and the handlers) at the recorded source revision in BASELINE and
# at the current tree, then compare the results test by test.
#
# A test that passed at the baseline and fails now is a regression, and the
# script exits 1. Tests that are new, or that failed at the baseline, are
# listed but do not fail the run.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tests/scripts/lib/common.sh"
HERE="${REPO}/backend/api/tests/scripts/aws-regression"
BASE="$(grep -v '^#' "${HERE}/BASELINE" | head -1 | tr -d '[:space:]')"
WT="$(mktemp -d)/baseline"

cleanup() { git -C "${REPO}" worktree remove --force "${WT}" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "== aws-regression: baseline ${BASE}, current $(git -C "${REPO}" rev-parse --short HEAD) plus the working tree"
git -C "${REPO}" worktree add --detach "${WT}" "${BASE}" >/dev/null

set +e
# CLUTTER_TF_VALIDATE=1 turns on the tests that run `terraform validate` on
# rendered AWS diagrams and round-trip the escaping through Terraform; they
# download the AWS provider once. LocalStack's apply test stays off unless
# CLUTTER_LOCALSTACK=1 is set and LocalStack is running.
export CLUTTER_TF_VALIDATE="${CLUTTER_TF_VALIDATE:-1}"
# Neither run has AWS credentials; the baseline is run the same way, so a
# test that needed AWS then shows as failing at the baseline.
env -u AWS_PROFILE -u AWS_ACCESS_KEY_ID -u AWS_SECRET_ACCESS_KEY \
  go -C "${WT}/backend/api" test -count=1 -json ./... >"${RESULTS}/aws-regression-baseline.jsonl" 2>"${RESULTS}/aws-regression-baseline.stderr"
env -u AWS_PROFILE -u AWS_ACCESS_KEY_ID -u AWS_SECRET_ACCESS_KEY \
  go -C "${REPO}/backend/api" test -count=1 -json ./... >"${RESULTS}/aws-regression-current.jsonl" 2>"${RESULTS}/aws-regression-current.stderr"
set -e

python3 "${REPO}/tests/scripts/lib/gotest_summary.py" aws-regression-baseline "${RESULTS}/aws-regression-baseline.jsonl" "${RESULTS}/aws-regression-baseline.json" || true
python3 "${REPO}/tests/scripts/lib/gotest_summary.py" aws-regression-current "${RESULTS}/aws-regression-current.jsonl" "${RESULTS}/aws-regression-current.json" || true
python3 "${HERE}/compare.py" "${BASE}" "${RESULTS}/aws-regression-baseline.json" "${RESULTS}/aws-regression-current.json" "${RESULTS}/aws-regression.json" "${HERE}/renames.tsv"
