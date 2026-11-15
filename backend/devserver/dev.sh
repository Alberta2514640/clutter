#!/usr/bin/env bash
# The one way to start Clutter locally (`npm run dev` at the repo root): A through the gateway, and B
# beside it. Development only. No AWS.
#   dev.sh                 checks the tools, starts Postgres and the backend (gateway and B API), then
#                          runs the frontend in this terminal. Ctrl+C stops everything.
#   dev.sh --backend-only  starts Postgres and the backend, then returns. Stop with stop.sh.
#   dev.sh --frontend-only runs only the website, against a backend already started with
#                          --backend-only. Ctrl+C stops the website alone.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

usage="usage: npm run dev | npm run dev:backend | npm run dev:frontend"
backend_only=false
frontend_only=false
case "$*" in
  "") ;;
  --backend-only) backend_only=true ;;
  --frontend-only) frontend_only=true ;;
  -h | --help | "--backend-only --help" | "--frontend-only --help")
    cat <<EOF
${usage}

Starts Clutter on this machine: Postgres in Docker, A's Lambdas behind a local API
Gateway, B API, and the website in this terminal. Ctrl+C stops it all and keeps your data.
  npm run dev:backend   (dev.sh --backend-only) starts everything but the website, then
                        returns; stop it with npm run stop
  npm run dev:frontend  (dev.sh --frontend-only) runs only the website, in this terminal,
                        against a backend started with npm run dev:backend
Run npm run setup once after cloning, and npm run doctor when a tool seems missing.
EOF
    exit 0
    ;;
  *) echo "${usage}" >&2; exit 2 ;;
esac

need() {
  command -v "$1" >/dev/null || { echo "Missing $1. $2 Run npm run doctor for the install command." >&2; exit 1; }
}
gateway_healthy() {
  curl -fs --max-time 1 "http://${API_ADDR}/dev/healthz" >/dev/null
}

need curl "The scripts need curl."
if ${frontend_only}; then
  if ! gateway_healthy || ! adapter_healthy; then
    echo "The backend is not running. Run npm run dev:backend first." >&2
    exit 1
  fi
else
  need go "Clutter needs Go."
  need docker "Clutter needs Docker."
  need openssl "The scripts need openssl."
  docker info >/dev/null 2>&1 || { echo "Docker is installed but not running. Open Docker Desktop (on Linux: sudo systemctl start docker), wait until it is ready, then run this again." >&2; exit 1; }
fi
if ! ${backend_only}; then
  need node "The website needs Node."
  [[ -d "${REPO_DIR}/frontend/clutter/node_modules" ]] || { echo "The frontend packages are missing. Run npm run setup first." >&2; exit 1; }
fi

# Each is true only while the recorded process runs (owned_pid in lib.sh).
gateway_process_alive() {
  owned_pid gateway "${STATE_DIR}/gateway" >/dev/null
}
adapter_process_alive() {
  owned_pid adapter "${STATE_DIR}/adapter-api" >/dev/null
}
website_process_alive() {
  owned_pid website npm >/dev/null
}

# Where to open the website, for both start messages.
website_lines() {
  cat <<EOF
  Open:     http://localhost:3000 (not the port 3100 address Next.js prints below)
  Sign in:  http://localhost:3000/dev/login signs you in as a local test user at once.
            Google sign-in at http://localhost:3000/login works too.
EOF
}

# The website, the one way both npm run dev and npm run dev:frontend start it. It
# runs in the background with its pid on disk, so `npm run stop` from another
# terminal (or a script) stops it too; Ctrl+C here runs the exit trap, which does
# the same. A website that exits with 130 or 143 (SIGINT or SIGTERM), or after
# stop.sh removed website.pid, stopped normally.
run_website() {
  local status=0 website_pid
  NEXT_PUBLIC_API_ENDPOINT="http://localhost:8787" NEXT_PUBLIC_ADAPTER_API_ENDPOINT="http://localhost:${ADAPTER_ADDR##*:}" NEXT_PUBLIC_GOOGLE_CLIENT_ID="214630517546-2tttdua57o7gj14up16v7s3unqoah46k.apps.googleusercontent.com" npm --prefix "${REPO_DIR}/frontend/clutter" run dev -- --hostname 127.0.0.1 --port 3100 &
  website_pid=$!
  write_pid_file website "${website_pid}" npm
  write_pid_file dev $$ dev.sh
  wait "${website_pid}" || status=$?
  [[ -f "${STATE_DIR}/website.pid" ]] || exit 0
  case "${status}" in
    130 | 143) exit 0 ;;
    *) exit "${status}" ;;
  esac
}

if ${frontend_only}; then
  if website_process_alive 2>/dev/null || curl -s -o /dev/null --max-time 1 "${FRONTEND_UPSTREAM}/"; then
    echo "The website already runs on ${FRONTEND_UPSTREAM}. Stop it first (npm run stop stops everything)." >&2
    exit 1
  fi
  # Ctrl+C here stops the website alone; the backend keeps running.
  stop_website() {
    local pid
    trap '' INT TERM
    trap - EXIT
    rm -f "${STATE_DIR}/dev.pid"
    if pid="$(owned_pid website npm 2>/dev/null)"; then
      rm -f "${STATE_DIR}/website.pid"
      stop_tree "${pid}"
    fi
  }
  trap stop_website EXIT
  trap 'stop_website; exit 0' INT TERM
  cat <<EOF
The website is starting against the backend on this machine. It is ready when Next.js prints "Ready" below.
$(website_lines)
  Stop:     Ctrl+C here stops the website. npm run stop stops the backend too.

EOF
  run_website
fi

if gateway_healthy || gateway_process_alive || adapter_healthy || adapter_process_alive || website_process_alive; then
  echo "Clutter is already running; restarting it so it picks up your code."
  # stop.sh's errors stay visible: it fails when an earlier dev.sh is still stopping.
  if ! "${DEVSERVER_DIR}/stop.sh" >/dev/null; then
    echo "Could not stop the running Clutter. Run npm run stop, then npm run dev again." >&2
    exit 1
  fi
fi
for addr in "${API_ADDR}" "${ADAPTER_ADDR}" "${FRONT_ADDR}"; do
  if curl -s -o /dev/null --max-time 1 "http://${addr}/"; then
    echo "Something already listens on ${addr}. Stop it first (lsof -nP -iTCP:${addr##*:} -sTCP:LISTEN)." >&2
    exit 1
  fi
done
if ! ${backend_only} && curl -s -o /dev/null --max-time 1 "${FRONTEND_UPSTREAM}/"; then
  echo "Something already listens on ${FRONTEND_UPSTREAM}. Stop it first (lsof -nP -iTCP:3100 -sTCP:LISTEN)." >&2
  exit 1
fi

cleanup() {
  trap '' INT TERM # a second Ctrl+C must not cut the stop short
  trap - EXIT
  echo
  echo "Stopping the website, the backend and the database (your data is kept)."
  # stop.sh waits for the dev.sh named in dev.pid; this one is already stopping.
  rm -f "${STATE_DIR}/dev.pid"
  "${DEVSERVER_DIR}/stop.sh" >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'cleanup; exit 0' INT TERM

# Runs a quiet step, and shows its output only when it fails.
quietly() {
  local out
  if ! out="$("$@" 2>&1)"; then
    echo "${out}" >&2
    exit 1
  fi
}

load_secrets
load_adapter_credential_key

echo "Starting the database and backend (the first run builds the Lambdas and takes a minute or two) ..."
quietly compose up -d --wait postgres
quietly go -C "${DEVSERVER_DIR}" build -o "${STATE_DIR}/gateway" ./cmd/gateway
quietly go -C "${REPO_DIR}/backend/adapter" build -o "${STATE_DIR}/adapter-api" ./cmd/api

# A clean environment: no AWS_* or other variables from this shell reach the
# gateway. Go needs PATH and HOME (for its caches).
nohup env -i \
  PATH="${PATH}" HOME="${HOME}" TMPDIR="${TMPDIR:-/tmp}" \
  ${GOPROXY:+GOPROXY="${GOPROXY}"} ${GOFLAGS:+GOFLAGS="${GOFLAGS}"} \
  JWT_SECRET="${JWT_SECRET}" \
  PSQL_CONNECTION_STRING="${PSQL_CONNECTION_STRING}" \
  "${STATE_DIR}/gateway" \
    -api-dir "${REPO_DIR}/backend/api" \
    -state-dir "${STATE_DIR}" \
    -api-addr "${API_ADDR}" \
    -front-addr "${FRONT_ADDR}" \
    -frontend-upstream "${FRONTEND_UPSTREAM}" \
  > "${STATE_DIR}/gateway.log" 2>&1 &
gateway_pid=$!
write_pid_file gateway "${gateway_pid}" "${STATE_DIR}/gateway"

# The pid is this script's own child, so kill -0 is enough here; its command
# line is still this script's until nohup starts.
for _ in $(seq 1 240); do
  gateway_healthy && break
  if ! kill -0 "${gateway_pid}" 2>/dev/null; then
    echo "The gateway exited. Last log lines:" >&2
    tail -n 30 "${STATE_DIR}/gateway.log" >&2
    exit 1
  fi
  sleep 0.5
done
if ! gateway_healthy; then
  echo "The gateway did not become ready. Last log lines:" >&2
  tail -n 30 "${STATE_DIR}/gateway.log" >&2
  exit 1
fi

# B API, with the same JWT secret as A (it verifies A's tokens) and only the
# variables it needs. It applies adapter-tables.sql at start-up. Connection
# credentials are encrypted under .state/adapter with the key kept in .state.
# When LocalStack runs (npm run localstack -- start), B's AWS connector gets
# its keys from LocalStack's STS and sends every run there, so an AWS deploy
# applies for real on this machine. Otherwise every AWS run stops at "needs a
# real AWS account".
adapter_aws=(ADAPTER_AWS_STS=local)
aws_mode="renders and previews AWS Terraform; runs need a real AWS account"
again="npm run dev"
${backend_only} && again="npm run dev:backend"
# The second line of the AWS start message: how to get real local AWS runs.
aws_next="For real AWS deploys here: npm run localstack -- start, then ${again} again (it restarts Clutter)."
if curl -fs --max-time 1 "${LOCALSTACK_ENDPOINT}/_localstack/health" >/dev/null; then
  adapter_aws=(ADAPTER_AWS_STS=localstack "ADAPTER_AWS_ENDPOINT=${LOCALSTACK_ENDPOINT}")
  aws_mode="AWS runs apply against LocalStack on ${LOCALSTACK_ENDPOINT}"
  aws_next=""
fi
nl=$'\n'
mkdir -p "${ADAPTER_DATA_DIR}"
chmod 700 "${ADAPTER_DATA_DIR}"
nohup env -i \
  PATH="${PATH}" TMPDIR="${TMPDIR:-/tmp}" \
  ADAPTER_LISTEN_ADDR="${ADAPTER_ADDR}" \
  ADAPTER_DATABASE_URL="${PSQL_CONNECTION_STRING}" \
  ADAPTER_SCHEMA_FILE="${REPO_DIR}/backend/storage-sql/adapter-tables.sql" \
  ADAPTER_DATA_DIR="${ADAPTER_DATA_DIR}" \
  JWT_SECRET="${JWT_SECRET}" \
  ADAPTER_CREDENTIAL_STORE=local \
  ADAPTER_CREDENTIAL_KEY="${ADAPTER_CREDENTIAL_KEY}" \
  ADAPTER_CONNECTORS=mock,general-openstack,aws \
  ADAPTER_AWS_TEMPLATE_DIR="${REPO_DIR}/backend/infra/modules/s3/uploads/aws" \
  "${adapter_aws[@]}" \
  ADAPTER_ALLOW_PRIVATE_TARGETS=1 \
  ADAPTER_ALLOWED_ORIGINS="http://localhost:3000,http://127.0.0.1:3000" \
  "${STATE_DIR}/adapter-api" \
  > "${STATE_DIR}/adapter.log" 2>&1 &
adapter_pid=$!
write_pid_file adapter "${adapter_pid}" "${STATE_DIR}/adapter-api"

for _ in $(seq 1 60); do
  adapter_healthy && break
  if ! kill -0 "${adapter_pid}" 2>/dev/null; then
    echo "B API exited. Last log lines:" >&2
    tail -n 30 "${STATE_DIR}/adapter.log" >&2
    exit 1
  fi
  sleep 0.5
done
if ! adapter_healthy; then
  echo "B API did not become ready. Last log lines:" >&2
  tail -n 30 "${STATE_DIR}/adapter.log" >&2
  exit 1
fi

if ${backend_only}; then
  trap - INT TERM EXIT
  cat <<EOF

The backend is up (no frontend).
  API:       http://localhost:8787
  B API:     http://localhost:${ADAPTER_ADDR##*:}   (connectors: mock, general-openstack, aws)
  AWS:       ${aws_mode}${aws_next:+${nl}             ${aws_next}}
  Website:   npm run dev:frontend, in another terminal
  Postgres:  127.0.0.1:${PG_PORT}, database clutter
  Logs:      ${STATE_DIR}/gateway.log, ${STATE_DIR}/adapter.log
  Sample:    npm run seed   (adds a sample AWS diagram for the local test user)
  Smoke:     npm run smoke
  Stop:      npm run stop   (add -- --wipe to delete the database)
EOF
  exit 0
fi

cat <<EOF

Clutter is starting. The website is ready when Next.js prints "Ready" below.
$(website_lines)
  Sample:   npm run seed, in another terminal, adds an organization, a project and an
            AWS diagram for the local test user. Each run adds a new set.
  B API:    http://localhost:${ADAPTER_ADDR##*:}
  AWS:      ${aws_mode}${aws_next:+${nl}            ${aws_next}}
  Logs:     ${STATE_DIR#"${REPO_DIR}"/}/gateway.log and adapter.log
  Stop:     Ctrl+C here (your data is kept)

EOF
run_website
