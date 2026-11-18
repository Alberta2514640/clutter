#!/usr/bin/env bash
# End-to-end and browser compatibility testing: the whole user workflow in
# Chromium, Firefox and WebKit, against the running app. Start it first with
# npm run dev. The first run installs Playwright's browsers (and, on Linux,
# their system libraries, which needs sudo).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"
HERE="${REPO}/tests/e2e-browser"
DEV="${REPO}/backend/devserver"
# shellcheck source=/dev/null
source "${DEV}/lib.sh"
WEB="${CLUTTER_WEB_URL:-http://localhost:3000}"
if ! curl -fs --max-time 5 -o /dev/null "${WEB}/login"; then
  echo "e2e-browser: the website is not answering at ${WEB}; start the app with npm run dev" | tee "${RESULTS}/e2e-browser.skipped"
  exit 2
fi
echo "== e2e-browser: ${WEB}"
npm --prefix "${HERE}" install --no-audit --no-fund --silent
if [[ "${E2E_INSTALL_DEPS:-1}" == 1 && "$(uname -s)" == Linux ]]; then
  (cd "${HERE}" && npx playwright install --with-deps chromium firefox webkit >/dev/null)
else
  (cd "${HERE}" && npx playwright install chromium firefox webkit >/dev/null)
fi
set +e
(cd "${HERE}" && CLUTTER_WEB_URL="${WEB}" CLUTTER_API_URL="http://${API_ADDR}" CLUTTER_ADAPTER_URL="http://${ADAPTER_ADDR}" \
  E2E_JSON="${RESULTS}/e2e-browser.json" E2E_ARTIFACTS="${RESULTS}/e2e-browser-artifacts" npx playwright test)
status=$?
set -e
exit "${status}"
