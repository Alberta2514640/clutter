#!/usr/bin/env bash
# Installs DevStack inside the Lima VM that `npm run devstack -- create` makes
# from lima.yaml. devstack.sh pipes this file to bash in the VM, as the VM's
# default user; it never runs on this machine.
#
# Safe to run again. It does nothing when DevStack is installed for the VM's
# current address, and installs DevStack again when that address changed (its
# catalog holds the address). It refuses a VM that lima.yaml did not make.
#
# The passwords in local.conf are random, made once inside the VM, and stay
# there (mode 600, owned by the stack user). Nothing secret is in this file.
set -euo pipefail

# Everything runs from main, so bash has read the whole script from stdin
# before any command can read stdin.
main() {
  local host_ip libvirt_type
  if [[ ! -f /etc/clutter-devstack ]]; then
    echo "This VM was not made from backend/devserver/devstack/lima.yaml, so DevStack is not installed here." >&2
    exit 1
  fi

  host_ip="$(ip -4 -o addr show dev lima0 2>/dev/null | awk '{ split($4, a, "/"); print a[1]; exit }')"
  if [[ -z "${host_ip}" ]]; then
    echo "The VM has no address on lima0, the vzNAT network lima.yaml asks for." >&2
    exit 1
  fi
  if [[ -e /dev/kvm ]]; then
    libvirt_type=kvm
  else
    libvirt_type=qemu
    echo "This VM has no /dev/kvm, so DevStack's own VMs will run under emulation (slow, but they work)."
  fi

  if ! id stack >/dev/null 2>&1; then
    sudo useradd -s /bin/bash -d /opt/stack -m stack
  fi
  sudo chmod 751 /opt/stack
  echo "stack ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/stack >/dev/null
  sudo chmod 440 /etc/sudoers.d/stack
  if ! command -v git >/dev/null 2>&1; then
    sudo apt-get update -q </dev/null >/dev/null
    sudo apt-get install -y -q git </dev/null >/dev/null
  fi

  sudo -u stack -i bash -s -- "${host_ip}" "${libvirt_type}" <<'AS_STACK'
set -euo pipefail
host_ip="$1"
libvirt_type="$2"
log=/opt/stack/stack.log
done_file=/opt/stack/.clutter-installed # the HOST_IP of the last stack.sh that completed

if [[ -f "${done_file}" && "$(cat "${done_file}")" == "${host_ip}" ]]; then
  echo "DevStack is already installed for ${host_ip}. Nothing to do."
  exit 0
fi

cd /opt/stack
if [[ ! -d devstack ]]; then
  echo "Cloning DevStack ..."
  git clone -q --depth 1 https://opendev.org/openstack/devstack
fi
cd devstack

if [[ ! -f local.conf ]]; then
  password="$(openssl rand -hex 16)"
  (
    umask 077
    printf '%s\n' \
      '[[local|localrc]]' \
      "ADMIN_PASSWORD=${password}" \
      "DATABASE_PASSWORD=${password}" \
      "RABBIT_PASSWORD=${password}" \
      "SERVICE_PASSWORD=${password}" \
      "HOST_IP=${host_ip}" \
      'GIT_DEPTH=1' \
      "LIBVIRT_TYPE=${libvirt_type}" \
      'disable_service tempest' >local.conf
  )
fi
sed -i -e "s/^HOST_IP=.*/HOST_IP=${host_ip}/" -e "s/^LIBVIRT_TYPE=.*/LIBVIRT_TYPE=${libvirt_type}/" local.conf

# An earlier run (finished for another address, or stopped part way) is
# removed first, so stack.sh starts clean.
if [[ -f "${log}" ]]; then
  echo "Removing the earlier DevStack install ..."
  ./unstack.sh >/opt/stack/unstack.log 2>&1 </dev/null || true
  ./clean.sh >/opt/stack/clean.log 2>&1 </dev/null || true
fi

rm -f "${done_file}"
echo "Installing DevStack for ${host_ip}. This takes 10 to 20 minutes."
echo "To follow it, run this in another terminal: limactl shell devstack tail -f ${log}"
rc=0
./stack.sh >"${log}" 2>&1 </dev/null || rc=$?
if [[ "${rc}" -ne 0 ]]; then
  echo "stack.sh failed (exit ${rc}). The end of ${log} in the VM:" >&2
  tail -n 40 "${log}" >&2
  exit 1
fi
printf '%s\n' "${host_ip}" >"${done_file}"
grep -E 'stack.sh completed in' "${log}" | tail -n 1 || true
echo "DevStack is installed for ${host_ip}."
AS_STACK
}

main "$@"
exit
