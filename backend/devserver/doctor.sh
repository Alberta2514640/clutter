#!/usr/bin/env bash
# Checks that this machine has what Clutter's local run needs (`npm run doctor`), one
# line per check, with the install command for this platform when something is
# missing. It installs nothing and never runs sudo. It exits with 1 only when a
# required tool is missing or unusable; warnings and optional tools never fail it.
# test-scripts.sh runs it against fake tools on PATH.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

case "$*" in
  "") ;;
  -h | --help)
    cat <<'EOF'
usage: npm run doctor

Checks the tools Clutter needs and prints one line per check:
  ok    found and new enough
  FAIL  a required tool is missing or unusable; the next line says what to do
  warn  works without it, but something will not (the next line says what)
  --    optional, or not checked yet
Exits with 1 when a FAIL line is printed, otherwise 0. It installs nothing.
EOF
    exit 0
    ;;
  *) echo "usage: npm run doctor" >&2; exit 2 ;;
esac

failures=0
warnings=0
GB=1048576 # KB in a GB
MIN_FREE_KB=$((5 * GB))

ok() { printf '  ok    %s\n' "$1"; }
fail() {
  printf '  FAIL  %s\n' "$1"
  if [[ -n "${2:-}" ]]; then printf '        %s\n' "$2"; fi
  failures=$((failures + 1))
}
warn() {
  printf '  warn  %s\n' "$1"
  if [[ -n "${2:-}" ]]; then printf '        %s\n' "$2"; fi
  warnings=$((warnings + 1))
}
note() {
  printf '  --    %s\n' "$1"
  if [[ -n "${2:-}" ]]; then printf '        %s\n' "$2"; fi
}
have() { command -v "$1" >/dev/null 2>&1; }

# version_ge A B: true when version A is B or newer. Compares up to three
# dot-separated numbers; anything after a number's digits (rc1, -rd) is ignored.
version_ge() {
  local i x y
  local -a a b
  IFS=. read -r -a a <<<"$1"
  IFS=. read -r -a b <<<"$2"
  for i in 0 1 2; do
    x="${a[i]:-0}"
    y="${b[i]:-0}"
    x="${x%%[!0-9]*}"
    y="${y%%[!0-9]*}"
    x="${x:-0}"
    y="${y:-0}"
    if ((10#${x} > 10#${y})); then return 0; fi
    if ((10#${x} < 10#${y})); then return 1; fi
  done
  return 0
}

# --- Platform, and the install command for each tool on it -------------------

os="$(uname -s)"
arch="$(uname -m)"
pkg=""
case "${os}" in
  Darwin)
    platform=mac
    if [[ "${arch}" == arm64 ]]; then platform_name="macOS, Apple Silicon"; else platform_name="macOS, Intel"; fi
    ;;
  Linux)
    platform=linux
    platform_name="Linux, ${arch}"
    if have apt-get; then pkg=apt; elif have dnf; then pkg=dnf; fi
    ;;
  *)
    platform=other
    platform_name="${os}, ${arch}"
    ;;
esac

# hint TOOL: what to run (or read) to install TOOL here.
hint() {
  case "${platform}:$1" in
    mac:node)
      if [[ -n "${NVM_DIR:-}" ]]; then echo "Install: nvm install (it reads .nvmrc)"; else echo "Install: brew install node"; fi
      ;;
    linux:node)
      if [[ -n "${NVM_DIR:-}" ]]; then
        echo "Install: nvm install (it reads .nvmrc)"
      else
        echo "Install Node $(node_want) or newer from https://nodejs.org/en/download (distro packages are often older)"
      fi
      ;;
    mac:go) echo "Install: brew install go" ;;
    linux:go) echo "Install Go from https://go.dev/doc/install (distro packages are often older)" ;;
    mac:docker) echo "Install: brew install --cask docker-desktop, then open Docker Desktop once (it adds docker to your PATH)" ;;
    linux:docker) echo "Install Docker Engine with its compose plugin: https://docs.docker.com/engine/install/" ;;
    mac:compose) echo "Docker Desktop includes it: brew install --cask docker-desktop" ;;
    linux:compose)
      case "${pkg}" in
        apt) echo "Install: sudo apt-get install docker-compose-plugin (from Docker's apt repository)" ;;
        dnf) echo "Install: sudo dnf install docker-compose-plugin (from Docker's dnf repository)" ;;
        *) echo "Install the compose plugin: https://docs.docker.com/compose/install/linux/" ;;
      esac
      ;;
    mac:docker-start) echo "Open Docker Desktop (open -a Docker) and wait until it says it is running." ;;
    linux:docker-start) echo "Start it: sudo systemctl start docker (and sudo systemctl enable docker to start it at boot)" ;;
    linux:docker-group) echo "Let your user run Docker: sudo usermod -aG docker \"\$USER\", then log out and back in (https://docs.docker.com/engine/install/linux-postinstall/)" ;;
    mac:terraform) echo "Install: brew tap hashicorp/tap && brew install hashicorp/tap/terraform" ;;
    linux:terraform) echo "Install from HashiCorp's apt or dnf repository: https://developer.hashicorp.com/terraform/install" ;;
    mac:python3) echo "Install: xcode-select --install (Apple's command line tools include python3)" ;;
    mac:curl | mac:openssl) echo "Install: brew install $1" ;;
    linux:curl | linux:openssl | linux:python3)
      case "${pkg}" in
        apt) echo "Install: sudo apt-get install $1" ;;
        dnf) echo "Install: sudo dnf install $1" ;;
        *) echo "Install $1 with your package manager." ;;
      esac
      ;;
    mac:lima) echo "Install: brew install lima" ;;
    linux:lima) echo "Install: https://lima-vm.io/docs/installation/" ;;
    mac:playwright) echo "Install: npx --prefix frontend/clutter playwright install chromium" ;;
    linux:playwright) echo "Install: npx --prefix frontend/clutter playwright install --with-deps chromium" ;;
    *:port) echo "See what holds it: lsof -nP -iTCP:$2 -sTCP:LISTEN (or ss -ltnp 'sport = :$2'), and stop it." ;;
    *) echo "Install $1 for ${os}; Clutter is tested on macOS and Linux." ;;
  esac
}

# --- What the repo asks for --------------------------------------------------

# Node: the major version .nvmrc names.
node_want() {
  local want=""
  if [[ -f "${REPO_DIR}/.nvmrc" ]]; then want="$(tr -dc '0-9.' <"${REPO_DIR}/.nvmrc")"; fi
  echo "${want}"
}

# Go: the newest `go` line among the repo's Go modules.
go_want() {
  local mod v want="0"
  for mod in "${REPO_DIR}"/backend/*/go.mod; do
    [[ -f "${mod}" ]] || continue
    v="$(awk '$1 == "go" { print $2; exit }' "${mod}")"
    if [[ -n "${v}" ]] && ! version_ge "${want}" "${v}"; then want="${v}"; fi
  done
  echo "${want}"
}

# Terraform: the version B's run image installs, so a local run matches the cloud.
terraform_want() {
  local file
  for file in "${REPO_DIR}"/backend/*/images/runtask.Dockerfile; do
    [[ -f "${file}" ]] || continue
    sed -n 's|^FROM hashicorp/terraform:\([0-9.]*\).*|\1|p' "${file}" | head -n 1
    return 0
  done
}

# free_kb DIR: free space, in KB, of the disk DIR is on.
free_kb() {
  df -Pk "$1" 2>/dev/null | awk 'NR == 2 { print $4 }'
}

# --- The checks --------------------------------------------------------------

echo "Checking what Clutter needs on this machine (${platform_name})."
echo
echo "Required:"

if [[ "${platform}" == other ]]; then
  warn "This platform is not tested. Clutter runs on macOS and Linux (Windows through WSL, untested)."
fi
if [[ "${platform}" == mac ]] && have sysctl && [[ "$(sysctl -n sysctl.proc_translated 2>/dev/null || true)" == 1 ]]; then
  warn "This terminal runs under Rosetta (Intel code on Apple Silicon)." \
    "Tools installed from it are Intel builds. Open a terminal without Rosetta, and use Homebrew in /opt/homebrew."
fi

# Node and npm.
want="$(node_want)"
if ! have node; then
  fail "Node: not found (needs ${want:-22} or newer)" "$(hint node)"
else
  got="$(node -v 2>/dev/null || true)"
  got="${got#v}"
  if [[ -z "${want}" ]]; then
    ok "Node ${got} (.nvmrc names no version, so it is not compared)"
  elif version_ge "${got}" "${want}"; then
    ok "Node ${got} (needs ${want} or newer, from .nvmrc)"
  else
    fail "Node ${got} is older than ${want} (.nvmrc)" "$(hint node)"
  fi
fi
if ! have npm; then
  fail "npm: not found (it comes with Node)" "$(hint node)"
else
  ok "npm $(npm -v 2>/dev/null || echo '(version unknown)')"
fi

# Go. Go 1.21 and newer download the toolchain a go.mod asks for, unless
# GOTOOLCHAIN is local (Fedora's packaged Go sets that).
want="$(go_want)"
if ! have go; then
  fail "Go: not found (needs ${want} or newer)" "$(hint go)"
else
  got="$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{ print $3 }' || true)"
  got="${got#go}"
  mode="$( (cd / && go env GOTOOLCHAIN) 2>/dev/null || true)"
  if version_ge "${got}" "${want}"; then
    ok "Go ${got} (needs ${want} or newer, from the go.mod files)"
  elif version_ge "${got}" 1.21 && [[ "${mode}" == auto || "${mode}" == *+auto ]]; then
    ok "Go ${got}; it downloads go${want} on the first build (GOTOOLCHAIN=${mode})"
  elif version_ge "${got}" 1.21; then
    fail "Go ${got} is older than ${want}, and GOTOOLCHAIN=${mode:-local} stops it downloading go${want}" \
      "Let Go download it with: go env -w GOTOOLCHAIN=auto. Or upgrade Go. $(hint go)"
  else
    fail "Go ${got} is older than ${want}" "$(hint go)"
  fi
fi

# Docker: installed, compose plugin, daemon reachable, and able to start a container.
docker_ok=false
if ! have docker; then
  fail "Docker: not found" "$(hint docker)"
else
  if docker compose version >/dev/null 2>&1; then
    compose_version="$(docker compose version --short 2>/dev/null || true)"
    ok "Docker Compose ${compose_version}"
  else
    fail "Docker Compose: the docker compose plugin is missing" "$(hint compose)"
  fi
  if info_err="$(docker info --format '{{.ServerVersion}}' 2>&1)"; then
    ok "Docker ${info_err}, running"
    docker_ok=true
  elif [[ "${info_err}" == *"ermission denied"* ]]; then
    if [[ "${platform}" == linux ]]; then
      fail "Docker refused this user (permission denied)" "$(hint docker-group)"
    else
      fail "Docker refused this user (permission denied)" "$(hint docker-start)"
    fi
  else
    fail "Docker is installed but not running" "$(hint docker-start)"
  fi
fi

docker_free=""
if ${docker_ok}; then
  pg_image="$(postgres_image || true)"
  if [[ -z "${pg_image}" ]]; then
    fail "Docker could not read backend/devserver/docker-compose.yml" "Run: docker compose -f backend/devserver/docker-compose.yml config, and read its error."
  elif docker image inspect "${pg_image}" >/dev/null 2>&1; then
    if run_out="$(docker_free_kb "${pg_image}" 2>&1)" && [[ "${run_out}" =~ ^[0-9]+$ ]]; then
      ok "docker run works for this user"
      docker_free="${run_out}"
    else
      fail "Docker could not start a container: $(printf '%s' "${run_out}" | head -n 1)" "$(hint docker-start)"
    fi
  else
    note "docker run: not checked yet, because the Postgres image is not pulled" "npm run setup pulls it and checks again."
  fi
  if [[ -z "${docker_free}" && "${platform}" == linux ]]; then
    root_dir="$(docker info --format '{{.DockerRootDir}}' 2>/dev/null || true)"
    if [[ -n "${root_dir}" ]]; then docker_free="$(free_kb "${root_dir}" || true)"; fi
  fi
fi

# The small tools the scripts call.
for tool in curl openssl; do
  if have "${tool}"; then ok "${tool}"; else fail "${tool}: not found" "$(hint "${tool}")"; fi
done

echo
echo "Recommended:"

if have python3; then
  ok "python3 ($(python3 --version 2>&1 | awk '{ print $2 }'))"
else
  warn "python3: not found; npm run seed needs it" "$(hint python3)"
fi

want="$(terraform_want || true)"
if ! have terraform; then
  warn "Terraform: not found; B's local deploys and npm run generated need it" "$(hint terraform)"
else
  got="$(terraform version 2>/dev/null | head -n 1 | awk '{ print $2 }' || true)"
  got="${got#v}"
  if [[ -z "${want}" ]]; then
    ok "Terraform ${got}"
  elif version_ge "${got}" "${want}"; then
    ok "Terraform ${got} (B's run image uses ${want})"
  else
    warn "Terraform ${got} is older than ${want}, which B's run image uses" "$(hint terraform)"
  fi
fi

# Ports. dev.sh stops and restarts its own backend, so a port Clutter's own
# stack holds is fine; anything else in the way stops npm run dev.
gateway_up=false
if curl -fs --max-time 1 "http://${API_ADDR}/dev/healthz" >/dev/null 2>&1; then gateway_up=true; fi
container_running() {
  ${docker_ok} && [[ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null || true)" == "true" ]]
}
port_busy() {
  local rc=0
  curl -s -o /dev/null --max-time 1 "http://127.0.0.1:$1/" >/dev/null 2>&1 || rc=$?
  [[ ${rc} -ne 7 ]] # 7: nothing listens
}
free_ports=""
our_ports=""
check_port() {
  local port="$1" what="$2" ours="$3"
  if ! port_busy "${port}"; then
    free_ports="${free_ports:+${free_ports}, }${port}"
  elif ${ours}; then
    our_ports="${our_ports:+${our_ports}, }${port}"
  else
    warn "port ${port} (${what}) is in use by another program; npm run dev needs it" "$(hint port "${port}")"
  fi
}
adapter_up=false
if adapter_healthy 2>/dev/null; then adapter_up=true; fi
pg_up=false
if container_running clutter-devserver-postgres; then pg_up=true; fi
check_port "${FRONT_ADDR##*:}" "the website" "${gateway_up}"
check_port "${FRONTEND_UPSTREAM##*:}" "Next.js" "${gateway_up}"
check_port "${API_ADDR##*:}" "A's API" "${gateway_up}"
check_port "${ADAPTER_ADDR##*:}" "B API" "${adapter_up}"
check_port "${PG_PORT}" "Postgres" "${pg_up}"
if [[ -n "${free_ports}" ]]; then ok "ports ${free_ports} are free"; fi
if [[ -n "${our_ports}" ]]; then
  note "ports ${our_ports} are held by Clutter's own running dev stack" \
    "Ctrl+C in the npm run dev terminal stops it, or npm run stop after npm run dev:backend."
fi

# Disk.
host_free="$(free_kb "${REPO_DIR}" || true)"
if [[ "${host_free}" =~ ^[0-9]+$ ]]; then
  if ((host_free < MIN_FREE_KB)); then
    warn "disk: $((host_free / GB)) GB free here; packages, Go's caches and Terraform's providers need about 5 GB"
  else
    ok "disk: $((host_free / GB)) GB free"
  fi
fi
if [[ "${docker_free}" =~ ^[0-9]+$ ]]; then
  if ((docker_free < MIN_FREE_KB)); then
    warn "Docker disk: $((docker_free / GB)) GB free; Postgres and LocalStack need a few GB" \
      "Free space with: docker system prune (it deletes stopped containers and unused images)"
  else
    ok "Docker disk: $((docker_free / GB)) GB free"
  fi
fi

echo
echo "Optional:"

if [[ "${platform}" == mac ]] && ! have brew; then
  note "Homebrew: not found; the install commands above use it" "Install it from https://brew.sh"
fi

# LocalStack: only for AWS runs on this machine.
if ! ${docker_ok}; then
  note "LocalStack image: not checked, because Docker is not usable"
elif docker image inspect "${LOCALSTACK_IMAGE}" >/dev/null 2>&1; then
  if container_running clutter-localstack; then
    ok "LocalStack: running on ${LOCALSTACK_ENDPOINT}"
  elif port_busy "${LOCALSTACK_ENDPOINT##*:}"; then
    warn "LocalStack image pulled, but port ${LOCALSTACK_ENDPOINT##*:} is in use by another program" "$(hint port "${LOCALSTACK_ENDPOINT##*:}")"
  else
    ok "LocalStack image pulled (start it with npm run localstack -- start)"
  fi
else
  note "LocalStack image: not pulled; only for AWS runs on this machine" "npm run localstack -- start pulls it the first time."
fi

# Lima and the DevStack VM: only for OpenStack runs on this machine.
# keystone_answers URL: true when Keystone answers at URL.
keystone_answers() {
  curl -fs -o /dev/null --max-time 3 "$1" >/dev/null 2>&1
}
if ! have limactl; then
  note "Lima: not found; only for DevStack (OpenStack on this machine)" "$(hint lima). Then follow ${DEVSTACK_GUIDE}."
else
  lima_version="$(limactl --version 2>/dev/null | awk '{ print $NF }' || true)"
  vm_status="$(limactl list "${DEVSTACK_VM}" --format '{{.Status}}' 2>/dev/null || true)"
  if [[ -z "${vm_status}" ]]; then
    note "Lima ${lima_version}; no VM named ${DEVSTACK_VM}" "Make it with npm run devstack -- create (${DEVSTACK_GUIDE})."
  else
    ok "Lima ${lima_version}; VM ${DEVSTACK_VM} is ${vm_status} (npm run devstack -- status)"
  fi
  # Clutter reaches DevStack at the VM's lima0 address, the HOST_IP its
  # catalog names. Lima also forwards the VM's ports to 127.0.0.1.
  if [[ "${vm_status}" == Running ]]; then
    vm_ip="$(limactl shell --workdir / "${DEVSTACK_VM}" ip -4 -o addr show dev lima0 2>/dev/null |
      awk '{ split($4, a, "/"); print a[1]; exit }' || true)"
    loopback_url="http://127.0.0.1/identity/v3"
    loopback_ok=false
    if keystone_answers "${loopback_url}"; then loopback_ok=true; fi
    if [[ -z "${vm_ip}" ]]; then
      warn "DevStack: the ${DEVSTACK_VM} VM has no lima0 address, so Clutter cannot reach DevStack" \
        "A VM made by npm run devstack -- create has one. See ${DEVSTACK_GUIDE}."
    elif keystone_answers "http://${vm_ip}/identity/v3"; then
      ok "DevStack answers at http://${vm_ip}/identity/v3 (the address Clutter uses)"
      if ${loopback_ok}; then
        ok "DevStack answers at ${loopback_url} too (Lima forwards the VM's ports)"
      else
        note "DevStack does not answer at ${loopback_url}; Clutter does not need it to"
      fi
    elif [[ "${platform}" == mac ]]; then
      if ${loopback_ok}; then
        warn "DevStack answers at ${loopback_url}, but not at http://${vm_ip}/identity/v3, where Clutter reaches it"
      else
        warn "DevStack does not answer at http://${vm_ip}/identity/v3 or ${loopback_url}" \
          "If DevStack is not installed yet, run npm run devstack -- create. Otherwise:"
      fi
      printf '        %s\n' \
        "macOS may be blocking this app from the VM. Open System Settings, Privacy & Security, Local Network," \
        "switch on the app you run this terminal or editor in (Terminal, iTerm, Cursor, VS Code), then try again."
    else
      warn "DevStack does not answer at http://${vm_ip}/identity/v3" \
        "Check that DevStack is installed in the VM (npm run devstack -- create) and that nothing blocks the VM's address."
    fi
  fi
fi

# Playwright's Chromium: only for browser tests.
if [[ -n "${PLAYWRIGHT_BROWSERS_PATH:-}" ]]; then
  pw_dir="${PLAYWRIGHT_BROWSERS_PATH}"
elif [[ "${platform}" == mac ]]; then
  pw_dir="${HOME}/Library/Caches/ms-playwright"
else
  pw_dir="${XDG_CACHE_HOME:-${HOME}/.cache}/ms-playwright"
fi
pw_found=false
for dir in "${pw_dir}"/chromium-* "${pw_dir}"/chromium_headless_shell-*; do
  if [[ -d "${dir}" ]]; then pw_found=true; fi
done
if ${pw_found}; then
  ok "Playwright's Chromium (for browser tests)"
else
  note "Playwright's Chromium: not installed; only for browser tests" "$(hint playwright)"
fi

echo
if ((failures > 0)); then
  echo "Not ready: ${failures} required check(s) failed. Fix the FAIL lines above, open a new terminal, then run npm run doctor again."
  exit 1
elif ((warnings > 0)); then
  echo "Ready, with ${warnings} warning(s) above."
else
  echo "Ready: everything Clutter needs is here."
fi
