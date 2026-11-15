#!/usr/bin/env bash
# Prepares a fresh clone for `npm run dev` (`npm run setup`). Safe to run again:
# each step skips what is already done. It runs doctor.sh first and stops when a
# required tool is missing; it installs no tools and never runs sudo.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

case "$*" in
  "") ;;
  -h | --help)
    cat <<'EOF'
usage: npm run setup

Prepares this clone for npm run dev. Safe to run again.
  1. npm run doctor: stops here when a required tool is missing
  2. the frontend's packages (npm ci in frontend/clutter), when package-lock.json changed
  3. every Go module's dependencies (go mod download), then one build to fill Go's cache
  4. the pinned Postgres image, and a check that Docker can start a container
  5. frontend/clutter/.env.local from .env.example, only when it does not exist
EOF
    exit 0
    ;;
  *) echo "usage: npm run setup" >&2; exit 2 ;;
esac

step() { printf '\n== %s\n' "$1"; }
fail() {
  echo "Setup stopped: $1" >&2
  exit 1
}

step "1/5 Checking the tools"
"${DEVSERVER_DIR}/doctor.sh" || fail "fix the FAIL lines above, open a new terminal, then run npm run setup again."

step "2/5 Frontend packages"
front="${REPO_DIR}/frontend/clutter"
if [[ -f "${front}/node_modules/.package-lock.json" && ! "${front}/package-lock.json" -nt "${front}/node_modules/.package-lock.json" ]]; then
  echo "Already installed (package-lock.json has not changed since)."
else
  echo "Installing the exact versions in frontend/clutter/package-lock.json (npm ci) ..."
  npm --prefix "${front}" ci --no-audit --no-fund || fail "npm ci failed; read its error above."
fi

step "3/5 Go modules"
modules=()
for mod in "${REPO_DIR}"/backend/*/go.mod; do
  if [[ -f "${mod}" ]]; then modules+=("$(dirname "${mod}")"); fi
done
((${#modules[@]} > 0)) || fail "found no go.mod under backend/."
echo "The first time, Go may download the toolchain the go.mod files ask for, then the modules."
for dir in "${modules[@]}"; do
  echo "Downloading ${dir#"${REPO_DIR}"/} modules ..."
  go -C "${dir}" mod download || fail "go mod download failed in ${dir#"${REPO_DIR}"/}; read its error above."
done
for dir in "${modules[@]}"; do
  echo "Building ${dir#"${REPO_DIR}"/} once, to fill Go's build cache ..."
  go -C "${dir}" build ./... || fail "the build failed in ${dir#"${REPO_DIR}"/}; read its error above."
done

step "4/5 Docker"
image="$(postgres_image || true)"
[[ -n "${image}" ]] || fail "Docker could not read backend/devserver/docker-compose.yml."
if docker image inspect "${image}" >/dev/null 2>&1; then
  echo "The Postgres image is already pulled."
else
  echo "Pulling the Postgres image (${image%%@*}) ..."
  docker pull --quiet "${image}" >/dev/null || fail "docker pull failed; check your network, then run npm run setup again."
fi
free="$(docker_free_kb "${image}")" || fail "Docker could not start a container from the Postgres image; read the error above."
echo "Docker can start containers ($((free / 1048576)) GB free for its images)."

step "5/5 Frontend settings"
if [[ -e "${front}/.env.local" ]]; then
  echo "frontend/clutter/.env.local already exists; left as it is."
else
  cp "${front}/.env.example" "${front}/.env.local"
  echo "Created frontend/clutter/.env.local from .env.example (npm run dev passes its own values over it)."
fi

echo
echo "Ready. Run: npm run dev"
