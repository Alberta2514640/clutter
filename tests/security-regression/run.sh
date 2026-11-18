#!/usr/bin/env bash
# Security regression testing:
#  1. one row per fixed System A defect (defects.tsv): its tests are run, and
#     the defect passes only when at least one test ran and none failed;
#  2. the customer role template has no wildcard Allow (check_role.py);
#  3. with the backend running (npm run dev:backend), the live cross-tenant
#     test (one user tries to read or change another organization's
#     projects, diagrams and Terraform through A's real Lambdas) and the
#     row-level security test (Postgres's anon role reads no tenant row).
# Writes tests/results/security-regression.json.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
HERE="${REPO}/tests/security-regression"
OUT="${RESULTS}/security-regression"
mkdir -p "${OUT}"
status=0

echo "== security-regression: fixed defects in A"
while IFS=$'\t' read -r id what module pkg regex; do
  [[ -z "${id}" || "${id}" == \#* ]] && continue
  set +e
  go -C "${REPO}/backend/${module}" test -count=1 -json -run "^(${regex})\$" "${pkg}" >"${OUT}/${id}.jsonl" 2>/dev/null
  set -e
  python3 "${REPO}/tests/lib/gotest_summary.py" "${id}" "${OUT}/${id}.jsonl" "${OUT}/${id}.json" >/dev/null || true
  python3 - "${OUT}/${id}.json" "${id}" "${what}" <<'PY' || status=1
import json, sys
s = json.load(open(sys.argv[1]))
c = s["counts"]
ran = c["pass"] + c["fail"]
verdict = "PASS" if ran and not c["fail"] and not s["failed_packages"] else ("MISSING" if not ran else "FAIL")
s["defect"] = {"id": sys.argv[2], "what": sys.argv[3], "verdict": verdict}
json.dump(s, open(sys.argv[1], "w"), indent=1)
print(f"  {sys.argv[2]} {verdict:7} {c['pass']:3} pass {c['fail']:2} fail  {sys.argv[3]}")
sys.exit(0 if verdict == "PASS" else 1)
PY
done <"${HERE}/defects.tsv"

echo "== security-regression: customer role"
python3 "${HERE}/check_role.py" "${REPO}/backend/infra/modules/s3/uploads/client_side_terraform_deployer_role.yaml" | tee "${OUT}/role.txt" || status=1

echo "== security-regression: live cross-tenant and row-level security"
DEV="${REPO}/backend/devserver"
if [[ -s "${DEV}/.state/pg_password" ]]; then
  # shellcheck source=/dev/null
  source "${DEV}/lib.sh"
  load_secrets
  if curl -fs --max-time 2 "http://${API_ADDR}/dev/healthz" >/dev/null; then
    set +e
    CLUTTER_API_URL="http://${API_ADDR}" PSQL_CONNECTION_STRING="${PSQL_CONNECTION_STRING}" JWT_SECRET="${JWT_SECRET}" \
      go -C "${DEV}" test -count=1 -v -json ./tests/crosstenant/ >"${OUT}/live.jsonl" 2>"${OUT}/live.stderr"
    set -e
    python3 "${REPO}/tests/lib/gotest_summary.py" security-live "${OUT}/live.jsonl" "${OUT}/live.json" || status=1
  else
    echo "  the gateway is not answering; live checks skipped" | tee "${OUT}/live.skipped"
  fi
else
  echo "  the backend is not running (npm run dev:backend); live checks skipped" | tee "${OUT}/live.skipped"
fi

python3 - "${OUT}" "${RESULTS}/security-regression.json" <<'PY'
import glob, json, os, sys
out, dst = sys.argv[1:]
defects = []
for p in sorted(glob.glob(os.path.join(out, "D*.json"))):
    s = json.load(open(p))
    d = s.get("defect", {})
    d["tests"] = [{"name": t["name"], "result": t["result"]} for t in s["tests"] if "/" not in t["name"]]
    defects.append(d)
live = json.load(open(os.path.join(out, "live.json"))) if os.path.exists(os.path.join(out, "live.json")) else None
json.dump({"defects": defects, "role": open(os.path.join(out, "role.txt")).read().strip(),
           "live": live, "live_skipped": os.path.exists(os.path.join(out, "live.skipped"))}, open(dst, "w"), indent=1)
PY
exit "${status}"
