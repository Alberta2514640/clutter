#!/usr/bin/env bash
# PostToolUse(Edit|Write): formats the one file just written, so diffs stay free of format noise.
# Go goes through gofmt and Terraform through `terraform fmt`. A *.tf.tmpl is skipped: Go template
# syntax is not valid HCL, and fmt would fail or mangle it. The frontend has no formatter wired yet.
# A file is formatted only if it is new (untracked) or was already formatted at HEAD. Many System A
# files are not (52 Go files use CRLF line endings, 7 Terraform files fail fmt), and formatting one
# on a small edit would rewrite every line of it and bury the real change.
set -uo pipefail
CLUTTER_HOOK_KIND=tidy # read by lib.sh: without jq, this hook passes instead of blocking
export CLUTTER_HOOK_KIND
. "$(dirname "$0")/lib.sh"

f="$(hook_field .tool_input.file_path)"
[ -f "$f" ] || exit 0

# Prints the file as committed at HEAD; fails if the file is untracked.
head_version() {
  local dir rel
  dir="$(cd "$(dirname "$f")" && pwd)" || return 1
  rel="$(git -C "$dir" ls-files --full-name -- "$(basename "$f")" 2>/dev/null)"
  [ -n "$rel" ] || return 1
  git -C "$dir" show "HEAD:$rel" 2>/dev/null
}

case "$f" in
  *.go)
    command -v gofmt >/dev/null 2>&1 || exit 0
    if old="$(head_version)"; then
      [ -z "$(printf '%s\n' "$old" | gofmt -l 2>/dev/null)" ] || exit 0
      printf '%s' "$old" | grep -q $'\r' && exit 0
    fi
    gofmt -w "$f" 2>/dev/null ;;
  *.tf)
    command -v terraform >/dev/null 2>&1 || exit 0
    if old="$(head_version)"; then
      printf '%s\n' "$old" | terraform fmt -check - >/dev/null 2>&1 || exit 0
    fi
    terraform fmt "$f" >/dev/null 2>&1 ;;
esac
exit 0
