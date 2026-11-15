#!/usr/bin/env bash
# Makes the OpenStack application credential Clutter uses for the DevStack in
# this VM. devstack.sh (`npm run devstack -- credential`) pipes this file to
# bash in the VM as the stack user; it never runs on this machine.
#
#   check   prints "exists" or "absent" for the credential named clutter
#   create  deletes that credential when it exists, makes a new one for the
#           demo user in the demo project, and prints what Clutter's
#           Connections page needs
#
# The secret goes to the terminal once and nowhere else: not to a file, and
# not to a log.
set -euo pipefail

NAME=clutter

main() {
  local mode="${1:-}" out id secret region auth_url
  if [[ ! -f /opt/stack/devstack/openrc ]]; then
    echo "DevStack is not installed in this VM. Run: npm run devstack -- create" >&2
    exit 1
  fi
  # openrc reads unset variables, so -u is off while it runs.
  set +u
  # shellcheck disable=SC1091 # DevStack's own file, inside the VM
  source /opt/stack/devstack/openrc demo demo >/dev/null 2>&1
  set -u

  case "${mode}" in
    check)
      if openstack application credential show "${NAME}" >/dev/null 2>&1; then echo exists; else echo absent; fi
      ;;
    create)
      if openstack application credential show "${NAME}" >/dev/null 2>&1; then
        openstack application credential delete "${NAME}"
        echo "Deleted the old ${NAME} credential."
      fi
      out="$(openstack application credential create "${NAME}" \
        --description "Clutter on this machine (npm run devstack -- credential)" -f json)"
      id="$(printf '%s' "${out}" | python3 -c 'import json, sys; print(json.load(sys.stdin)["id"])')"
      secret="$(printf '%s' "${out}" | python3 -c 'import json, sys; print(json.load(sys.stdin)["secret"])')"
      region="$(openstack region list -f value -c Region | head -n 1)"
      # openrc points OS_AUTH_URL at Keystone on HOST_IP, the VM's lima0
      # address, which this machine reaches.
      auth_url="${OS_AUTH_URL%/}"
      [[ "${auth_url}" == */v3 ]] || auth_url="${auth_url}/v3"
      cat <<EOF

Made the application credential "${NAME}" for the demo user in the demo project.
In Clutter, open Organization settings, then Connections, and choose OpenStack.
Fill in the form with these values:

  Name                           any name, for example devstack
  Region                         ${region}
  Auth URL                       ${auth_url}
  Application credential ID      ${id}
  Application credential secret  ${secret}

The secret is shown only this once, and is not saved anywhere. Copy it now.
To get a new one, run npm run devstack -- credential again; the old one then
stops working, and Clutter needs the new values.
EOF
      ;;
    *)
      echo "usage: credential.sh check|create" >&2
      exit 2
      ;;
  esac
}

main "$@"
exit
