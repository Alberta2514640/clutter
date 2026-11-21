#!/usr/bin/env bash
# Starts, stops or reports on LocalStack, which stands in for AWS on this machine
# (`npm run localstack -- start|stop|status`). Development and tests only.
#
# Only the frozen Apache-2.0 image localstack/localstack:4.14.0 is used, pinned
# by digest in lib.sh (LOCALSTACK_IMAGE). It needs no auth token; never point
# this at a newer tag, which does (docs/research/aws/2026-11-13-localstack.md).
# It listens on loopback only, keeps nothing between starts, and mounts the
# Docker socket so Lambda can start function containers.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

IMAGE="${LOCALSTACK_IMAGE}"
CONTAINER="clutter-localstack"

usage() {
  cat <<EOF
usage: npm run localstack -- start|stop|status

LocalStack stands in for AWS on this machine, so B's AWS deployments apply for real.
  start   starts LocalStack 4.14.0 on ${LOCALSTACK_ENDPOINT} (the first start pulls the image)
  stop    removes the container; LocalStack keeps nothing between starts
  status  says whether it runs, and lists its services

Start it before npm run dev, which checks for it once at start.
EOF
}

healthy() {
  curl -fs --max-time 2 "${LOCALSTACK_ENDPOINT}/_localstack/health" >/dev/null
}

running() {
  [[ "$(docker inspect -f '{{.State.Running}}' "${CONTAINER}" 2>/dev/null || true)" == "true" ]]
}

start() {
  command -v docker >/dev/null || { echo "Missing docker. Install Docker Desktop: https://www.docker.com/products/docker-desktop" >&2; exit 1; }
  command -v curl >/dev/null || { echo "Missing curl. Install curl with your package manager." >&2; exit 1; }
  docker info >/dev/null 2>&1 || { echo "Docker is installed but not running. Open Docker Desktop, wait until it is ready, then run this again." >&2; exit 1; }
  if running && healthy; then
    echo "LocalStack is already running on ${LOCALSTACK_ENDPOINT}."
    return 0
  fi
  if curl -s -o /dev/null --max-time 1 "${LOCALSTACK_ENDPOINT}/"; then
    echo "Something else already listens on ${LOCALSTACK_ENDPOINT#http://}. Stop it first (lsof -nP -iTCP:4566 -sTCP:LISTEN)." >&2
    exit 1
  fi
  docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
  echo "Starting LocalStack 4.14.0 on ${LOCALSTACK_ENDPOINT} (the first start pulls the image) ..."
  # No LOCALSTACK_AUTH_TOKEN: the pinned image runs without one.
  docker run -d --rm --name "${CONTAINER}" \
    -p "127.0.0.1:4566:4566" \
    -e DEBUG=0 \
    -e PERSISTENCE=0 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    "${IMAGE}" >/dev/null
  for _ in $(seq 1 120); do
    healthy && break
    if ! running; then
      echo "LocalStack exited while starting." >&2
      exit 1
    fi
    sleep 0.5
  done
  if ! healthy; then
    echo "LocalStack did not become ready. Last log lines:" >&2
    docker logs --tail 30 "${CONTAINER}" >&2 || true
    exit 1
  fi
  echo "LocalStack is ready on ${LOCALSTACK_ENDPOINT}. Stop it with: npm run localstack -- stop"
}

stop() {
  if docker inspect "${CONTAINER}" >/dev/null 2>&1; then
    echo "Stopping LocalStack (it keeps no data) ..."
    docker rm -f "${CONTAINER}" >/dev/null
    echo "LocalStack is stopped."
  else
    echo "LocalStack is not running."
  fi
}

status() {
  if running && healthy; then
    echo "LocalStack is running on ${LOCALSTACK_ENDPOINT}."
    curl -fs --max-time 2 "${LOCALSTACK_ENDPOINT}/_localstack/health"
    echo
  else
    echo "LocalStack is not running."
    return 1
  fi
}

case "${1:-}" in
  start) start ;;
  stop) stop ;;
  status) status ;;
  -h | --help) usage ;;
  *) usage >&2; exit 2 ;;
esac
