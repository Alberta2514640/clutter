#!/usr/bin/env bash
# shellcheck disable=SC2016 # the fake tools' bodies expand their variables when they run, not here
# Checks the devserver's own scripts without touching Docker, Go or the network:
# doctor.sh and devstack.sh run against fake tools on a PATH that holds nothing
# else, and every script's --help answers before it does any work.
# Run: backend/devserver/test-scripts.sh (CI runs it too). VERBOSE=1 shows every
# run's output, not only a failing one's.
set -euo pipefail
DEVSERVER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASH_BIN="$(command -v bash)"

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
stubs="${work}/bin"
home="${work}/home"
mkdir -p "${home}"
passed=0
failed=0

# The real commands the scripts may call besides the tools under test.
real_tools=(bash env awk sed grep head df dirname cat tr)

# stub NAME BODY: a fake command that runs BODY.
stub() {
  printf '#!%s\n%s\n' "${BASH_BIN}" "$2" >"${stubs}/$1"
  chmod +x "${stubs}/$1"
}

# fresh_stubs: a PATH with every required tool present and working.
fresh_stubs() {
  local tool
  rm -rf "${stubs}"
  mkdir -p "${stubs}"
  for tool in "${real_tools[@]}"; do
    ln -s "$(command -v "${tool}")" "${stubs}/${tool}"
  done
  stub uname 'case "$1" in -s) echo "${FAKE_OS:-Darwin}" ;; -m) echo "${FAKE_ARCH:-arm64}" ;; esac'
  stub node 'echo "${FAKE_NODE:-v22.12.0}"'
  stub npm 'echo 10.9.0'
  stub go 'case "$1" in version) echo "go version go${FAKE_GO:-1.25.3} test/test" ;; env) echo "${FAKE_GOTOOLCHAIN:-auto}" ;; esac'
  stub terraform 'echo "Terraform v${FAKE_TF:-1.14.4}"'
  stub openssl 'echo "OpenSSL 3.0.0"'
  stub python3 'echo "Python 3.12.0"'
  stub curl 'exit 7' # nothing listens anywhere
  stub docker '
case "$1" in
  --version) echo "Docker version 29.0.0" ;;
  compose)
    case "$*" in
      *"config --images"*) echo "postgres:test" ;;
      *version*) echo "2.30.0" ;;
    esac ;;
  info)
    case "${FAKE_DOCKER:-running}" in
      stopped) echo "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?" >&2; exit 1 ;;
      denied) echo "permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock" >&2; exit 1 ;;
    esac
    case "$*" in *DockerRootDir*) echo / ;; *) echo 29.0.0 ;; esac ;;
  image) [[ "$3" == postgres:test ]] ;;
  run)
    [[ "${FAKE_DOCKER_RUN:-ok}" == ok ]] || { echo "docker: Error response from daemon: failed to create task" >&2; exit 125; }
    printf "Filesystem 1024-blocks Used Available Capacity Mounted on\noverlay 100000000 1 99999999 1%% /\n" ;;
  *) exit 1 ;;
esac'
}

out="${work}/out"
rc=0
# run SCRIPT [VAR=value ...]: runs a script with the stub PATH and a clean environment.
run() {
  local script="$1"
  shift
  rc=0
  env -i PATH="${stubs}" HOME="${home}" "$@" "${BASH_BIN}" "${DEVSERVER_DIR}/${script}" >"${out}" 2>&1 || rc=$?
}
run_args() {
  local script="$1"
  shift
  rc=0
  env -i PATH="${stubs}" HOME="${home}" "${BASH_BIN}" "${DEVSERVER_DIR}/${script}" "$@" >"${out}" 2>&1 || rc=$?
}

# check NAME EXIT [TEXT ...]: the last run exited with EXIT and printed every TEXT.
check() {
  local name="$1" want="$2" text problem=""
  shift 2
  [[ "${rc}" == "${want}" ]] || problem="exit ${rc}, want ${want}"
  for text in "$@"; do
    grep -qF -- "${text}" "${out}" || problem="${problem:+${problem}; }missing: ${text}"
  done
  if [[ -z "${problem}" ]]; then
    passed=$((passed + 1))
    echo "pass  ${name}"
    if [[ -n "${VERBOSE:-}" ]]; then sed 's/^/        | /' "${out}"; fi
  else
    failed=$((failed + 1))
    echo "FAIL  ${name}: ${problem}"
    sed 's/^/        | /' "${out}"
  fi
}
# refute NAME TEXT: the last run did not print TEXT.
refute() {
  if grep -qF -- "$2" "${out}"; then
    failed=$((failed + 1))
    echo "FAIL  $1: printed: $2"
  else
    passed=$((passed + 1))
    echo "pass  $1"
  fi
}

echo "doctor.sh"
fresh_stubs
run doctor.sh
check "all ok" 0 "ok    Node 22.12.0" "ok    npm 10.9.0" "ok    Go 1.25.3" "ok    Docker 29.0.0, running" \
  "ok    docker run works for this user" "ok    Terraform 1.14.4" "Ready"
refute "all ok has no FAIL line" "FAIL"

fresh_stubs
rm "${stubs}/go"
run doctor.sh FAKE_OS=Darwin
check "Go missing on macOS" 1 "FAIL  Go: not found (needs 1.25.3 or newer)" "brew install go" "Not ready: 1 required check(s) failed"

fresh_stubs
rm "${stubs}/go"
stub apt-get 'exit 0'
run doctor.sh FAKE_OS=Linux FAKE_ARCH=x86_64
check "Go missing on Linux" 1 "(Linux, x86_64)" "FAIL  Go: not found" "https://go.dev/doc/install"

fresh_stubs
run doctor.sh FAKE_GO=1.22.2 FAKE_GOTOOLCHAIN=auto
check "old Go that downloads the toolchain" 0 "ok    Go 1.22.2; it downloads go1.25.3"

fresh_stubs
run doctor.sh FAKE_GO=1.22.2 FAKE_GOTOOLCHAIN=local
check "old Go with GOTOOLCHAIN=local" 1 "FAIL  Go 1.22.2 is older than 1.25.3" "go env -w GOTOOLCHAIN=auto"

fresh_stubs
run doctor.sh FAKE_NODE=v20.11.0
check "Node too old" 1 "FAIL  Node 20.11.0 is older than 22" "brew install node"

fresh_stubs
run doctor.sh FAKE_DOCKER=stopped FAKE_OS=Darwin
check "Docker stopped on macOS" 1 "FAIL  Docker is installed but not running" "open -a Docker"

fresh_stubs
run doctor.sh FAKE_DOCKER=stopped FAKE_OS=Linux
check "Docker stopped on Linux" 1 "FAIL  Docker is installed but not running" "sudo systemctl start docker"

fresh_stubs
run doctor.sh FAKE_DOCKER=denied FAKE_OS=Linux
check "Docker refuses this user on Linux" 1 "permission denied" "sudo usermod -aG docker"

fresh_stubs
run doctor.sh FAKE_DOCKER_RUN=broken
check "docker run fails" 1 "FAIL  Docker could not start a container"

fresh_stubs
rm "${stubs}/docker"
run doctor.sh FAKE_OS=Darwin
check "Docker missing on macOS" 1 "FAIL  Docker: not found" "brew install --cask docker-desktop"

fresh_stubs
rm "${stubs}/terraform"
run doctor.sh FAKE_OS=Darwin
check "Terraform missing only warns" 0 "warn  Terraform: not found" "brew install hashicorp/tap/terraform" "Ready, with"

fresh_stubs
rm "${stubs}/terraform"
stub dnf 'exit 0'
run doctor.sh FAKE_OS=Linux
check "Terraform missing on Linux" 0 "https://developer.hashicorp.com/terraform/install"

fresh_stubs
run doctor.sh
check "optional tools absent are notes" 0 "--    Lima: not found" "--    Playwright's Chromium: not installed" "--    LocalStack image: not pulled" \
  "OpenStack on this machine (DevStack)"

# A running devstack VM at 192.168.64.2. FAKE_KEYSTONE is curl's exit code for
# the VM's address, FAKE_LOOPBACK for 127.0.0.1; everything else has nothing listening.
fresh_stubs
stub limactl '
case "$1" in
  --version) echo "limactl version 2.3.0" ;;
  list) echo Running ;;
  shell) echo "3: lima0    inet 192.168.64.2/24 metric 100 brd 192.168.64.255 scope global dynamic lima0" ;;
esac'
stub curl '
case "$*" in
  *192.168.64.2/identity*) exit "${FAKE_KEYSTONE:-0}" ;;
  *127.0.0.1/identity*) exit "${FAKE_LOOPBACK:-0}" ;;
  *) exit 7 ;;
esac'
run doctor.sh
check "DevStack reachable" 0 "ok    Lima 2.3.0; VM devstack is Running" \
  "ok    DevStack answers at http://192.168.64.2/identity/v3" "ok    DevStack answers at http://127.0.0.1/identity/v3 too"
refute "DevStack reachable has no Local Network hint" "Local Network"

run doctor.sh FAKE_KEYSTONE=28 FAKE_OS=Darwin
check "DevStack VM address blocked on macOS" 0 \
  "warn  DevStack answers at http://127.0.0.1/identity/v3, but not at http://192.168.64.2/identity/v3" \
  "Privacy & Security, Local Network" "Cursor, VS Code" "Ready, with"

run doctor.sh FAKE_KEYSTONE=7 FAKE_LOOPBACK=7 FAKE_OS=Darwin
check "DevStack not answering on macOS" 0 "warn  DevStack does not answer at http://192.168.64.2/identity/v3 or" \
  "npm run devstack -- create" "Local Network"

run doctor.sh FAKE_KEYSTONE=7 FAKE_OS=Linux
check "DevStack not answering on Linux" 0 "warn  DevStack does not answer at http://192.168.64.2/identity/v3"
refute "no macOS hint on Linux" "Local Network"

echo
echo "devstack.sh"
fresh_stubs
run_args devstack.sh status
check "Lima missing" 1 "Lima is not installed" "OpenStack on this machine (DevStack)"

stub limactl 'echo "time=x level=fatal msg=\"unmatched instances\"" >&2; exit 1'
run_args devstack.sh start
check "no devstack VM" 1 "There is no Lima VM named devstack" "npm run devstack -- create"

stub limactl 'case "$*" in *--format*) echo Stopped ;; list*) echo "NAME STATUS"; echo "devstack Stopped" ;; esac'
run_args devstack.sh status
check "VM stopped" 1 "devstack Stopped"

# create refuses a devstack VM it did not make, before it changes anything:
# this limactl fails the test if it is asked to create, start or shell.
stub limactl '
case "$*" in
  *ClutterDevstack*) echo "" ;;
  *--format*) echo Running ;;
  create* | start* | shell*) echo "UNEXPECTED limactl $*" ; exit 1 ;;
esac'
run_args devstack.sh create
check "create refuses a VM it did not make" 1 "A Lima VM named devstack exists, and create did not make it" "limactl delete devstack"
refute "create changed nothing" "UNEXPECTED"

rc=0
env -i PATH="${stubs}" HOME="${home}" FAKE_OS=Linux "${BASH_BIN}" "${DEVSERVER_DIR}/devstack.sh" create >"${out}" 2>&1 || rc=$?
check "create refuses outside macOS" 1 "create needs macOS"
refute "create on Linux changed nothing" "UNEXPECTED"

# create with no VM: this limactl records its calls in calls.log and reads
# what create pipes into the VM.
calls="${work}/calls.log"
stub limactl '
echo "limactl $*" >>"'"${calls}"'"
case "$*" in
  *--format*) exit 1 ;;
  shell*) cat >/dev/null; echo "DevStack is installed for 192.168.64.2." ;;
esac'
stub sw_vers 'echo "${FAKE_MACOS:-27.2}"'
stub sysctl 'echo "${FAKE_CHIP:-Apple M4 Pro}"'
# create_with CHIP: runs devstack.sh create on a Mac with that chip, then adds
# limactl's calls to the output.
create_with() {
  rm -f "${calls}"
  rc=0
  env -i PATH="${stubs}" HOME="${home}" FAKE_CHIP="$1" "${BASH_BIN}" "${DEVSERVER_DIR}/devstack.sh" create >"${out}" 2>&1 || rc=$?
  if [[ -f "${calls}" ]]; then cat "${calls}" >>"${out}"; fi
}
create_with "Apple M4 Pro"
check "create makes, starts and installs on an M4" 0 "limactl create --tty=false --name devstack" "devstack/lima.yaml" \
  "limactl start --tty=false devstack" "limactl shell --workdir / devstack bash -s" "npm run devstack -- credential"
refute "an M4 keeps nested virtualization" "nestedVirtualization = false"

create_with "Apple M1"
check "create on an M1 turns nested virtualization off" 0 "--set .nestedVirtualization = false" "under emulation"

run_args devstack.sh credential
check "credential refuses output that is not a terminal" 1 "runs only when its output is a terminal"
refute "credential called nothing" "UNEXPECTED"

run_args devstack.sh status extra
check "devstack.sh refuses an extra argument" 2 "usage: npm run devstack"

echo
echo "--help"
fresh_stubs
for script in doctor.sh setup.sh dev.sh stop.sh smoke.sh localstack.sh devstack.sh generated.sh; do
  run_args "${script}" --help
  check "${script} --help" 0 "usage: npm run"
done
run_args doctor.sh --nonsense
check "doctor.sh refuses an unknown flag" 2 "usage: npm run doctor"
run_args dev.sh --frontend-only
check "dev.sh --frontend-only with no backend" 1 "Run npm run dev:backend first."
if command -v python3 >/dev/null 2>&1; then
  rc=0
  python3 "${DEVSERVER_DIR}/seed-diagram.py" --help >"${out}" 2>&1 || rc=$?
  check "seed-diagram.py --help" 0 "usage: npm run seed"
else
  echo "skip  seed-diagram.py --help (no python3 here)"
fi

echo
echo "${passed} passed, ${failed} failed"
((failed == 0))
