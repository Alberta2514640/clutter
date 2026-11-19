#!/usr/bin/env bash
# End-to-end and browser compatibility testing: the whole user workflow in
# Chromium, Firefox and WebKit, against the running app. Start it first with
# npm run dev. Uses the frontend's own Playwright (npm ci in frontend/clutter
# first); the first run installs Playwright's browsers (and, on Linux, their
# system libraries, which needs sudo). See frontend/clutter/tests/README.md.
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tests/scripts/lib/common.sh"
FRONTEND="${REPO}/frontend/clutter"
# shellcheck source=/dev/null
source "${REPO}/backend/devserver/lib.sh"
WEB="${CLUTTER_WEB_URL:-http://localhost:3000}"
if ! curl -fs --max-time 5 -o /dev/null "${WEB}/login"; then
  echo "e2e-browser: the website is not answering at ${WEB}; start the app with npm run dev" | tee "${RESULTS}/e2e-browser.skipped"
  exit 2
fi
[[ -d "${FRONTEND}/node_modules/@playwright/test" ]] || npm --prefix "${FRONTEND}" ci --no-audit --no-fund
echo "== e2e-browser: ${WEB}"
if [[ "${E2E_INSTALL_DEPS:-1}" == 1 && "$(uname -s)" == Linux ]]; then
  (cd "${FRONTEND}" && npx --no-install playwright install --with-deps chromium firefox webkit >/dev/null)
else
  (cd "${FRONTEND}" && npx --no-install playwright install chromium firefox webkit >/dev/null)
fi
set +e
(cd "${FRONTEND}" && CLUTTER_WEB_URL="${WEB}" CLUTTER_API_URL="http://${API_ADDR}" CLUTTER_ADAPTER_URL="http://${ADAPTER_ADDR}" \
  E2E_JSON="${RESULTS}/e2e-browser.json" E2E_ARTIFACTS="${RESULTS}/e2e-browser-artifacts" \
  npx --no-install playwright test -c tests/e2e/playwright.config.ts "$@")
status=$?
set -e
exit "${status}"
