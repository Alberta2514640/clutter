#!/usr/bin/env bash
# Stops the website, B API, the gateway, any Lambda process left behind, and Postgres.
#   --wipe  also deletes the Postgres volume (the next dev.sh reloads the schema),
#           the local S3 files and B's data (run logs and connection credentials),
#           which belong to the rows in it
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

wipe=false
case "$*" in
  "") ;;
  --wipe) wipe=true ;;
  -h | --help)
    cat <<'EOF'
usage: npm run stop [-- --wipe]

Stops the website, B API, the gateway, any Lambda process left behind, and Postgres.
Your data is kept.
  --wipe  also deletes the local database, the local S3 files and B's data, to start clean.
          The next npm run dev loads the schema again.
EOF
    exit 0
    ;;
  *) echo "usage: npm run stop [-- --wipe]" >&2; exit 2 ;;
esac

# stop_process LABEL NAME WORD SECONDS: when .state/NAME.pid names its process still
# (owned_pid in lib.sh), SIGTERM, wait up to SECONDS, then SIGKILL if that same process
# still runs.
stop_process() {
  local label="$1" name="$2" word="$3" seconds="$4" pid start
  pid="$(owned_pid "${name}" "${word}")" || return 0
  start="$(process_start "${pid}")"
  echo "Stopping ${label} (pid ${pid}) ..."
  kill -TERM "${pid}" 2>/dev/null || true
  for _ in $(seq 1 $((seconds * 2))); do
    [[ "$(process_start "${pid}")" == "${start}" ]] || break
    sleep 0.5
  done
  if [[ "$(process_start "${pid}")" == "${start}" ]]; then
    kill -KILL "${pid}" 2>/dev/null || true
  fi
  rm -f "${STATE_DIR}/${name}.pid"
}

# The website runs under dev.sh, which records both pids. Stop the website, then
# wait for dev.sh to finish its own exit trap (it runs this script too), so the
# two stops do not overlap: that trap stops B API (up to 35 seconds), the gateway
# (up to 10) and Postgres, and a late `compose down` from it would stop the
# Postgres a new dev.sh had started. owned_pid keeps a reused pid safe.
dev_pid=""
dev_start=""
if dev_pid="$(owned_pid dev dev.sh)"; then
  dev_start="$(process_start "${dev_pid}")"
fi
if pid="$(owned_pid website npm)"; then
  rm -f "${STATE_DIR}/website.pid"
  echo "Stopping the website (pid ${pid}) ..."
  stop_tree "${pid}"
fi
if [[ -n "${dev_start}" ]]; then
  # Up to 3 minutes: the trap's slowest stop is about a minute.
  for _ in $(seq 1 360); do
    [[ "$(process_start "${dev_pid}")" == "${dev_start}" ]] || break
    sleep 0.5
  done
  if [[ "$(process_start "${dev_pid}")" == "${dev_start}" ]]; then
    echo "dev.sh (pid ${dev_pid}) is still stopping after 3 minutes. Run npm run stop again once it has finished." >&2
    exit 1
  fi
fi
rm -f "${STATE_DIR}/dev.pid"

# The live sync service, when dev.sh started it. It keeps nothing on disk.
stop_process "the live sync service" collab "${COLLAB_DIR}/src/server.ts" 5

# B finishes or fails its runs in progress within 30 seconds of SIGTERM.
stop_process "B API" adapter "${STATE_DIR}/adapter-api" 35
stop_process "the gateway" gateway "${STATE_DIR}/gateway" 10

# Lambda processes outlive a gateway that was killed hard; match them by path.
if pgrep -f "${STATE_DIR}/bin/" >/dev/null 2>&1; then
  echo "Stopping leftover Lambda processes ..."
  pkill -f "${STATE_DIR}/bin/" || true
fi
rm -f "${STATE_DIR}/lambda.pids"

# Compose needs the password variable set to parse the file, even to stop.
if [[ -s "${STATE_DIR}/pg_password" ]]; then
  load_secrets
else
  export CLUTTER_DEV_PG_PASSWORD=unused
fi
if ${wipe}; then
  echo "Stopping Postgres and deleting its volume, the local S3 files and B's data ..."
  compose down -v
  rm -rf "${STATE_DIR}/s3" "${ADAPTER_DATA_DIR}"
else
  echo "Stopping Postgres (data kept; use --wipe to delete it) ..."
  compose down
fi
echo "Clutter is stopped: the website, A's API, B API and Postgres."
