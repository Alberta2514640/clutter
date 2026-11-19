#!/usr/bin/env bash
# Platform and version check: the toolchains and images are pinned, the pins
# agree, CI builds on them, and this machine meets them. CI runs this on
# every pull request (the checks workflow's platform job).
set -euo pipefail
source "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tests/scripts/lib/common.sh"
echo "== platform-version"
python3 "${REPO}/tests/scripts/platform-version/check_versions.py" "${REPO}" "${RESULTS}/platform-version.json"
