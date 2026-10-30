#!/usr/bin/env bash
# PostToolUse(Edit|Write): formats the one file just written, so diffs stay free of format noise.
# Go goes through gofmt and Terraform through `terraform fmt`. A *.tf.tmpl is skipped: Go template
# syntax is not valid HCL, and fmt would fail or mangle it. The frontend has no formatter wired yet.
set -uo pipefail
. "$(dirname "$0")/lib.sh"

f="$(hook_field .tool_input.file_path)"
[ -f "$f" ] || exit 0
case "$f" in
  *.go) command -v gofmt >/dev/null 2>&1 && gofmt -w "$f" 2>/dev/null ;;
  *.tf) command -v terraform >/dev/null 2>&1 && terraform fmt "$f" >/dev/null 2>&1 ;;
esac
exit 0
