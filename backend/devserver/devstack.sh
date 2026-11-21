#!/usr/bin/env bash
# Makes, starts, stops or reports on the Lima VM that runs DevStack, a small
# OpenStack for B's OpenStack connector, makes the application credential
# Clutter uses, says how to sign in to DevStack's dashboard, and routes
# DevStack's public network to the VM so its servers open from this machine
# (`npm run devstack -- create|credential|dashboard|start|stop|status|route`).
# The VM's template and the scripts it runs inside the VM are in devstack/.
# Development only. The guide is DEVSTACK_GUIDE (lib.sh).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

usage() {
  cat <<EOF
usage: npm run devstack -- create|credential|dashboard|start|stop|status
       npm run devstack -- route [remove] [--dry-run]

DevStack (OpenStack on this machine) runs in a Lima VM named ${DEVSTACK_VM}.
  create      makes the VM from backend/devserver/devstack/lima.yaml and
              installs DevStack in it (10 to 20 minutes the first time).
              Safe to run again. It refuses a VM named ${DEVSTACK_VM} that it
              did not make. macOS only.
  credential  makes the application credential Clutter uses, replacing the old
              one, and prints the values for Clutter's Connections page. The
              secret is shown once, so it runs only in a terminal.
  dashboard   prints the address of DevStack's dashboard and the demo login.
              The password is printed only to a terminal.
  start       starts the VM, and sets up DevStack's public network in it
              (${DEVSTACK_PUBLIC_RANGE}, again at every boot)
  stop        stops the VM (its disk, and DevStack in it, are kept)
  status      shows the VM's state; exits with 1 when it is not running
  route       sends ${DEVSTACK_PUBLIC_RANGE}, where DevStack's servers get their
              addresses, to the VM, so they open from this machine. It changes
              this machine's routes with sudo, which asks for your password;
              nothing keeps it. Run it again after this machine restarts.
              remove takes the route out; --dry-run prints the command only.
              macOS and Linux.

The guide is ${DEVSTACK_GUIDE}.
EOF
}

case "${1:-}" in
  create | credential | dashboard | start | stop | status | route) action="$1" ;;
  -h | --help) usage; exit 0 ;;
  *) usage >&2; exit 2 ;;
esac
shift
route_remove=false
dry_run=false
if [[ "${action}" == route ]]; then
  for arg in "$@"; do
    case "${arg}" in
      remove) route_remove=true ;;
      --dry-run) dry_run=true ;;
      *) usage >&2; exit 2 ;;
    esac
  done
elif [[ $# -gt 0 ]]; then
  usage >&2
  exit 2
fi

# The secret goes to the terminal only: never to a file, a pipe or a log.
if [[ "${action}" == credential && ! -t 1 ]]; then
  echo "credential prints a secret, so it runs only when its output is a terminal." >&2
  echo "Run npm run devstack -- credential in a terminal, without a pipe or a redirect." >&2
  exit 1
fi

if [[ "${action}" == create && "$(uname -s)" != Darwin ]]; then
  echo "create needs macOS: the VM's address comes from Apple's virtualization network (vzNAT)." >&2
  echo "start, stop, status and credential work with a ${DEVSTACK_VM} VM made another way." >&2
  exit 1
fi

if [[ "${action}" == route && "$(uname -s)" != Darwin && "$(uname -s)" != Linux ]]; then
  echo "route works on macOS and Linux only (this is $(uname -s))." >&2
  exit 1
fi

# run_route ARGS...: prints one route command, then runs it with sudo, which
# asks for the password itself. A dry run only prints it. With no terminal to
# type into (an editor's or agent's shell), sudo cannot ask, so on macOS the
# system password box asks instead. The arguments are fixed words and a
# checked IPv4 address, so they hold no quote to break the AppleScript string.
run_route() {
  echo "  sudo $*"
  if ${dry_run}; then return 0; fi
  if [[ "$(uname -s)" == Darwin && ! -t 0 ]] && ! sudo -n true >/dev/null 2>&1; then
    echo "  (no terminal here, so macOS asks for your password in a window)"
    osascript -e "do shell script \"$*\" with prompt \"Clutter wants to change this Mac's routes so it can open the servers on local OpenStack.\" with administrator privileges" >/dev/null
    return
  fi
  sudo "$@"
}

route_intro() {
  if ${dry_run}; then
    echo "Dry run: nothing is changed. This would run:"
  else
    echo "Changing this machine's routes. You are asked for your password, and nothing keeps it."
  fi
}

# remove_route: takes out this machine's own route for DEVSTACK_PUBLIC_RANGE,
# whatever it goes through. It needs neither Lima nor the VM.
remove_route() {
  local via
  via="$(devstack_route_via)"
  if [[ -z "${via}" ]]; then
    echo "This machine has no route of its own for ${DEVSTACK_PUBLIC_RANGE}, so there is nothing to remove."
    return 0
  fi
  route_intro
  case "$(uname -s)" in
    Darwin) run_route route -n delete -net "${DEVSTACK_PUBLIC_RANGE}" ;;
    Linux) run_route ip route del "${DEVSTACK_PUBLIC_RANGE}" ;;
  esac
  if ! ${dry_run}; then
    echo "Removed the route through ${via}. This machine sends ${DEVSTACK_PUBLIC_RANGE} to its default gateway again."
  fi
}

# add_route: sends DEVSTACK_PUBLIC_RANGE to the VM's lima0 address, replacing
# a route through another address (the VM's old one, for example).
add_route() {
  local via vm_ip
  if [[ "${vm_status}" != Running ]]; then
    echo "The ${DEVSTACK_VM} VM is ${vm_status}. Start it first: npm run devstack -- start" >&2
    exit 1
  fi
  vm_ip="$(devstack_vm_address)"
  if [[ ! "${vm_ip}" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
    echo "The ${DEVSTACK_VM} VM has no lima0 address (got: ${vm_ip:-nothing}), so there is nothing to route to. See ${DEVSTACK_GUIDE}." >&2
    exit 1
  fi
  via="$(devstack_route_via)"
  if [[ "${via}" == "${vm_ip}" ]]; then
    echo "This machine already sends ${DEVSTACK_PUBLIC_RANGE} to the ${DEVSTACK_VM} VM at ${vm_ip}. Nothing to do."
    return 0
  fi
  route_intro
  case "$(uname -s)" in
    Darwin)
      if [[ -n "${via}" ]]; then run_route route -n delete -net "${DEVSTACK_PUBLIC_RANGE}"; fi
      run_route route -n add -net "${DEVSTACK_PUBLIC_RANGE}" "${vm_ip}"
      ;;
    Linux) run_route ip route replace "${DEVSTACK_PUBLIC_RANGE}" via "${vm_ip}" ;;
  esac
  if ! ${dry_run}; then
    if [[ -n "${via}" ]]; then echo "Replaced the old route through ${via}."; fi
    echo "This machine now sends ${DEVSTACK_PUBLIC_RANGE} to the ${DEVSTACK_VM} VM at ${vm_ip}, so DevStack servers' addresses open here."
    echo "The route lasts until this machine restarts; then run npm run devstack -- route again."
  fi
}

if [[ "${action}" == route ]] && ${route_remove}; then
  remove_route
  exit 0
fi

if ! command -v limactl >/dev/null 2>&1; then
  case "$(uname -s)" in
    Darwin) install="brew install lima" ;;
    *) install="see https://lima-vm.io/docs/installation/" ;;
  esac
  echo "Lima is not installed, and DevStack runs in a Lima VM named ${DEVSTACK_VM}." >&2
  echo "Install Lima (${install}), then follow ${DEVSTACK_GUIDE}." >&2
  exit 1
fi

vm_status="$(limactl list "${DEVSTACK_VM}" --format '{{.Status}}' 2>/dev/null || true)"

# in_vm ARGS...: runs a command in the VM from its root folder, so Lima does not
# try this machine's current folder, which the VM does not have.
in_vm() {
  limactl shell --workdir / "${DEVSTACK_VM}" "$@"
}

# nested_virt_supported: true on Apple M3 or newer with macOS 15 or newer, where
# Lima's VZ engine can let DevStack's own VMs use hardware virtualization.
nested_virt_supported() {
  local chip major re='^Apple M([0-9]+)'
  chip="$(sysctl -n machdep.cpu.brand_string 2>/dev/null || true)"
  major="$(sw_vers -productVersion 2>/dev/null || true)"
  major="${major%%.*}"
  [[ "${chip}" =~ ${re} ]] || return 1
  ((BASH_REMATCH[1] >= 3)) && [[ "${major}" =~ ^[0-9]+$ ]] && ((major >= 15))
}

create_vm() {
  local marker
  local -a args=(create --tty=false --name "${DEVSTACK_VM}")
  if [[ -n "${vm_status}" ]]; then
    marker="$(limactl list "${DEVSTACK_VM}" --format '{{index .Config.Param "ClutterDevstack"}}' 2>/dev/null || true)"
    if [[ "${marker}" != 1 ]]; then
      echo "A Lima VM named ${DEVSTACK_VM} exists, and create did not make it, so it is left alone." >&2
      echo "Keep using it (npm run devstack -- start, credential), or delete it with" >&2
      echo "limactl delete ${DEVSTACK_VM} (this deletes its disk) and run create again." >&2
      exit 1
    fi
    echo "The ${DEVSTACK_VM} VM exists (${vm_status}), made by create."
    if [[ "${vm_status}" != Running ]]; then
      echo "Starting it ..."
      limactl start --tty=false "${DEVSTACK_VM}"
    fi
  else
    if ! nested_virt_supported; then
      args+=(--set '.nestedVirtualization = false')
      echo "This Mac cannot give the VM nested virtualization (it needs Apple M3 or newer and macOS 15 or newer)."
      echo "DevStack will work, but its own VMs will run under emulation, which is much slower."
    fi
    echo "Making the ${DEVSTACK_VM} VM (4 CPUs, 12 GiB of memory, a 40 GiB disk) ..."
    limactl "${args[@]}" "${DEVSTACK_DIR}/lima.yaml"
    echo "Starting it (the first start downloads Ubuntu 24.04) ..."
    limactl start --tty=false "${DEVSTACK_VM}"
  fi
  in_vm bash -s <"${DEVSTACK_DIR}/install.sh"
  apply_network
  cat <<EOF

DevStack runs in the ${DEVSTACK_VM} VM. Next:
  npm run doctor                    checks that this machine reaches DevStack
  npm run devstack -- credential    makes the credential to paste into Clutter
  npm run devstack -- route         lets this machine open DevStack servers' addresses
                                    (asks for your password)
EOF
}

# apply_network: sets up DevStack's public network in the VM, now and at every
# boot (devstack/network.sh). Safe to run again.
apply_network() {
  in_vm bash -s -- "${DEVSTACK_PUBLIC_RANGE}" "${DEVSTACK_PUBLIC_GATEWAY}" <"${DEVSTACK_DIR}/network.sh"
}

# route_hint: says how to open DevStack servers from this machine when its
# route for DevStack's public network does not go to the VM.
route_hint() {
  local vm_ip
  [[ "$(uname -s)" == Darwin || "$(uname -s)" == Linux ]] || return 0
  vm_ip="$(devstack_vm_address)"
  if [[ -n "${vm_ip}" && "$(devstack_route_via)" != "${vm_ip}" ]]; then
    echo "To open DevStack servers' addresses from this machine, run npm run devstack -- route (asks for your password)."
  fi
}

make_credential() {
  local state answer
  state="$(in_vm sudo -u stack -i bash -s -- check <"${DEVSTACK_DIR}/credential.sh")"
  if [[ "${state}" == exists ]]; then
    echo "DevStack already has an application credential named clutter."
    echo "Replacing it stops every Clutter connection that uses it until you give Clutter the new values."
    printf 'Replace it? [y/N] '
    answer=""
    read -r answer || true
    case "${answer}" in
      y | Y | yes | YES) ;;
      *) echo "Kept the old credential."; exit 0 ;;
    esac
  elif [[ "${state}" != absent ]]; then
    echo "Could not read DevStack's application credentials (got: ${state:-nothing})." >&2
    exit 1
  fi
  in_vm sudo -u stack -i bash -s -- create <"${DEVSTACK_DIR}/credential.sh"
}

# show_dashboard prints how to sign in to DevStack's dashboard. The password
# goes to a terminal only, as credential's secret does: never to a pipe or a
# file.
show_dashboard() {
  local address password
  address="$(in_vm sudo -u stack -i bash -s -- address <"${DEVSTACK_DIR}/dashboard.sh")"
  echo "Dashboard  ${address}"
  echo "Domain     Default"
  echo "User       demo (project demo)"
  if [[ -t 1 ]]; then
    password="$(in_vm sudo -u stack -i bash -s -- password <"${DEVSTACK_DIR}/dashboard.sh")"
    echo "Password   ${password}"
  else
    echo "Password   shown only in a terminal: run npm run devstack -- dashboard without a pipe or a redirect"
  fi
}

if [[ "${action}" == create ]]; then
  create_vm
  exit 0
fi

if [[ -z "${vm_status}" ]]; then
  echo "There is no Lima VM named ${DEVSTACK_VM}. Make it with npm run devstack -- create, as ${DEVSTACK_GUIDE} says." >&2
  exit 1
fi

case "${action}" in
  credential)
    if [[ "${vm_status}" != Running ]]; then
      echo "The ${DEVSTACK_VM} VM is ${vm_status}. Start it first: npm run devstack -- start" >&2
      exit 1
    fi
    make_credential
    ;;
  dashboard)
    if [[ "${vm_status}" != Running ]]; then
      echo "The ${DEVSTACK_VM} VM is ${vm_status}. Start it first: npm run devstack -- start" >&2
      exit 1
    fi
    show_dashboard
    ;;
  start)
    if [[ "${vm_status}" == Running ]]; then
      echo "The ${DEVSTACK_VM} VM is already running."
    else
      echo "Starting the ${DEVSTACK_VM} VM ..."
      limactl start "${DEVSTACK_VM}"
      echo "The ${DEVSTACK_VM} VM is running. Open a shell in it with: limactl shell ${DEVSTACK_VM}"
    fi
    apply_network
    route_hint
    ;;
  route) add_route ;;
  stop)
    if [[ "${vm_status}" == Running ]]; then
      echo "Stopping the ${DEVSTACK_VM} VM (its disk is kept) ..."
      limactl stop "${DEVSTACK_VM}"
    else
      echo "The ${DEVSTACK_VM} VM is not running (${vm_status})."
    fi
    ;;
  status)
    limactl list "${DEVSTACK_VM}"
    [[ "${vm_status}" == Running ]] || exit 1
    ;;
esac
