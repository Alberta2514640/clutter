#!/usr/bin/env bash
# Reads what a person needs to sign in to DevStack's dashboard (Horizon) in
# this VM. devstack.sh (`npm run devstack -- dashboard`) pipes this file to
# bash in the VM as the stack user; it never runs on this machine.
#
#   address   prints the dashboard's address, from local.conf's HOST_IP
#   password  prints the demo user's password: local.conf's ADMIN_PASSWORD,
#             which DevStack gives the demo user too
#
# devstack.sh asks for the password only when its own output is a terminal.
set -euo pipefail

CONF=/opt/stack/devstack/local.conf

# value NAME: the value of NAME= in local.conf.
value() {
  sed -n "s/^$1=//p" "${CONF}" | head -n 1
}

main() {
  local mode="${1:-}" v
  if [[ ! -f "${CONF}" ]]; then
    echo "DevStack is not installed in this VM. Run: npm run devstack -- create" >&2
    exit 1
  fi
  case "${mode}" in
    address) v="$(value HOST_IP)" ;;
    password) v="$(value ADMIN_PASSWORD)" ;;
    *)
      echo "usage: dashboard.sh address|password" >&2
      exit 2
      ;;
  esac
  if [[ -z "${v}" ]]; then
    echo "local.conf has no value for ${mode}." >&2
    exit 1
  fi
  if [[ "${mode}" == address ]]; then
    echo "http://${v}/dashboard"
  else
    echo "${v}"
  fi
}

main "$@"
