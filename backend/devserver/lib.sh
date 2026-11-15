# shellcheck shell=bash disable=SC2034  # sourced: its variables are used by the scripts that source it
# Shared settings for the scripts here (dev.sh, stop.sh, smoke.sh, doctor.sh, setup.sh and the
# rest). Sourced, not run. Bash 3.2 (macOS's own) must be able to run them.

DEVSERVER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${DEVSERVER_DIR}/../.." && pwd)"
STATE_DIR="${DEVSERVER_DIR}/.state"
COMPOSE_FILE="${DEVSERVER_DIR}/docker-compose.yml"

API_ADDR="127.0.0.1:8787"
ADAPTER_ADDR="127.0.0.1:8790"
ADAPTER_DATA_DIR="${STATE_DIR}/adapter"
# The live sync service (backend/collaboration). It is optional: Clutter runs without it.
COLLAB_ADDR="127.0.0.1:8791"
COLLAB_DIR="${REPO_DIR}/backend/collaboration"
FRONT_ADDR="127.0.0.1:3000"
FRONTEND_UPSTREAM="http://127.0.0.1:3100"
PG_PORT="54329"
# LocalStack (localstack.sh), when it runs. dev.sh points B's AWS connector at
# it, and the tests read the same address.
LOCALSTACK_ENDPOINT="http://127.0.0.1:4566"
# The only LocalStack image Clutter uses: the frozen Apache-2.0 4.14.0, pinned by
# its multi-arch index digest (amd64 and arm64), read from Docker Hub on
# 2026-11-13. It needs no auth token; never point this at a newer tag, which
# does (docs/research/aws/2026-11-13-localstack.md).
LOCALSTACK_IMAGE="localstack/localstack:4.14.0@sha256:3ebc37595918b8accb852f8048fef2aff047d465167edd655528065b07bc364a"
# DevStack runs in a Lima VM of this name (devstack.sh), made from the files in
# DEVSTACK_DIR, as this guide says.
DEVSTACK_VM="devstack"
DEVSTACK_DIR="${DEVSERVER_DIR}/devstack"
DEVSTACK_GUIDE="the \"OpenStack on this machine (DevStack)\" section of backend/devserver/README.md"

# Generates a secret once and keeps it in .state (mode 600).
local_secret() {
  local file="${STATE_DIR}/$1" prefix="$2"
  if [[ ! -s "${file}" ]]; then
    umask 077
    printf '%s%s' "${prefix}" "$(openssl rand -hex 32)" > "${file}"
  fi
  cat "${file}"
}

load_secrets() {
  mkdir -p "${STATE_DIR}"
  chmod 700 "${STATE_DIR}"
  CLUTTER_DEV_PG_PASSWORD="$(local_secret pg_password '')"
  JWT_SECRET="$(local_secret jwt_secret 'devserver-local-')"
  PSQL_CONNECTION_STRING="postgres://clutter:${CLUTTER_DEV_PG_PASSWORD}@127.0.0.1:${PG_PORT}/clutter?sslmode=disable"
  export CLUTTER_DEV_PG_PASSWORD CLUTTER_DEV_PG_PORT="${PG_PORT}"
}

# B's local credential key: 32 random bytes in base64, generated once and kept
# in .state (mode 600). It encrypts the connection credentials under .state/adapter.
load_adapter_credential_key() {
  local file="${STATE_DIR}/adapter_credential_key"
  if [[ ! -s "${file}" ]]; then
    (umask 077 && openssl rand -base64 32 > "${file}")
  fi
  ADAPTER_CREDENTIAL_KEY="$(cat "${file}")"
}

adapter_healthy() {
  curl -fs --max-time 1 "http://${ADAPTER_ADDR}/healthz" >/dev/null
}

collab_healthy() {
  curl -fs --max-time 1 "http://${COLLAB_ADDR}/healthz" >/dev/null
}

compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

# Pid files. dev.sh records each process it starts in .state/NAME.pid as three lines: the
# pid, the process's start time (ps -o lstart=, which macOS and Linux both print) and a word
# its command line holds. dev.sh and stop.sh signal a pid only while both still match, so a
# pid that the system gave to another process after a crash or a reboot is left alone.

# process_start PID: prints PID's start time, or nothing when no process has that pid.
process_start() {
  local out
  out="$(ps -o lstart= -p "$1" 2>/dev/null)" || return 0
  printf '%s\n' "${out}" | awk '{ $1 = $1; print }'
}

# write_pid_file NAME PID WORD: records PID in .state/NAME.pid with its start time and WORD.
write_pid_file() {
  printf '%s\n%s\n%s\n' "$2" "$(process_start "$2")" "$3" > "${STATE_DIR}/$1.pid"
}

# owned_pid NAME WORD: prints the pid in .state/NAME.pid when that process still runs, with
# the recorded start time and the recorded word in its command line. Otherwise it removes the
# file, says why on stderr, and fails. A one-line file, from a dev.sh that recorded no start
# time, is checked against WORD alone.
owned_pid() {
  local file="${STATE_DIR}/$1.pid" pid="" start="" word="" now="" cmd=""
  [[ -f "${file}" ]] || return 1
  { IFS= read -r pid || true; IFS= read -r start || true; IFS= read -r word || true; } < "${file}"
  [[ -n "${word}" ]] || word="$2"
  if [[ "${pid}" =~ ^[0-9]+$ ]]; then
    now="$(process_start "${pid}")"
    cmd="$(ps -o command= -p "${pid}" 2>/dev/null || true)"
    if [[ -n "${now}" && "${cmd}" == *"${word}"* ]] && [[ -z "${start}" || "${now}" == "${start}" ]]; then
      echo "${pid}"
      return 0
    fi
  fi
  rm -f "${file}"
  if [[ -n "${now}" ]]; then
    echo "Removed .state/$1.pid: pid ${pid} now belongs to another program, which is left alone." >&2
  else
    echo "Removed .state/$1.pid: pid ${pid:-?} no longer runs." >&2
  fi
  return 1
}

# stop_tree PID: SIGTERM to PID and every process under it (npm, Next.js and its
# workers), wait up to 10 seconds, then SIGKILL what is left.
stop_tree() {
  local pids=("$1") i=0 child
  while ((i < ${#pids[@]})); do
    for child in $(pgrep -P "${pids[i]}" 2>/dev/null); do pids+=("${child}"); done
    i=$((i + 1))
  done
  kill -TERM "${pids[@]}" 2>/dev/null || true
  for _ in $(seq 1 20); do
    kill -0 "${pids[@]}" 2>/dev/null || return 0
    sleep 0.5
  done
  kill -KILL "${pids[@]}" 2>/dev/null || true
}

# The Postgres image docker-compose.yml pins, read from the file so it has one owner.
postgres_image() {
  CLUTTER_DEV_PG_PASSWORD="${CLUTTER_DEV_PG_PASSWORD:-unused}" compose config --images 2>/dev/null | head -n 1
}

# docker_free_kb IMAGE: starts a throwaway container from IMAGE (already pulled),
# with no network, and prints the free space of Docker's storage in KB. It fails
# when Docker cannot start a container for this user.
docker_free_kb() {
  docker run --rm --pull never --network none --entrypoint df "$1" -Pk / | awk 'NR == 2 { print $4 }'
}
