#!/usr/bin/env bash
# Shared helpers for the Clutter hooks. Plain bash 3.2 (the macOS default) plus jq.
# Every hook reads the Claude Code hook payload (JSON) on stdin.

# hook_field <jq path>: prints one field of the payload, or nothing.
hook_field() {
  printf '%s' "$HOOK_RAW" | jq -r "$1 // empty" 2>/dev/null || true
}

# deny <reason>: refuse the tool call and tell the model why.
deny() {
  jq -cn --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
  exit 0
}

# ask <reason>: hand the call to the person to approve, whatever the permission mode.
ask() {
  jq -cn --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"ask",permissionDecisionReason:$r}}'
  exit 0
}

# split_commands <command line>: prints each chained command on its own line (&&, ||, ;, |, &).
# Crude by design: it does not parse quotes, so it can over-split, which only makes a guard stricter.
split_commands() {
  printf '%s\n' "$1" | awk '{ gsub(/&&|\|\||;|\||&/, "\n"); print }'
}

# Word splitting below must never expand a `*` in a command into file names.
set -f

# A hook that cannot parse its payload lets the call through rather than blocking all work.
if ! command -v jq >/dev/null 2>&1; then
  echo "clutter hooks: jq is not installed, so the guards are off. Install it with: brew install jq" >&2
  exit 0
fi

HOOK_RAW="$(cat)"
