#!/usr/bin/env bash
# Makes the OpenStack application credential Clutter uses for the DevStack in
# this VM. devstack.sh (`npm run devstack -- credential`) pipes this file to
# bash in the VM as the stack user; it never runs on this machine.
#
#   check   prints "exists" or "absent" for the credential named clutter
#   create  deletes that credential when it exists, makes a new one for the
#           demo user in the demo project, and prints what Clutter's
#           Connections page needs
#   link NAME    makes a new credential named NAME (clutter-<8 hex>) for the
#                demo user in the demo project, and prints its id, secret and
#                region as one JSON line. It never replaces a credential. The
#                devserver gateway runs it for the website's "Use this
#                machine's DevStack" button and hands the line to B.
#   remove NAME  deletes the credential NAME (clutter-<8 hex>) when it exists
#
# The secret goes to the terminal once (create) or to the gateway once (link),
# and nowhere else: not to a file, and not to a log.
set -euo pipefail

NAME=clutter

main() {
  local mode="${1:-}" name="${2:-}" out id secret region auth_url
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
    link | remove)
      if [[ ! "${name}" =~ ^clutter-[0-9a-f]{8}$ ]]; then
        echo "usage: credential.sh ${mode} clutter-<8 hex digits>" >&2
        exit 2
      fi
      ;;
  esac

  case "${mode}" in
    link)
      if openstack application credential show "${name}" >/dev/null 2>&1; then
        echo "A credential named ${name} exists already; it is left alone." >&2
        exit 1
      fi
      region="$(openstack region list -f value -c Region | head -n 1)"
      if [[ -z "${region}" ]]; then
        echo "DevStack lists no region." >&2
        exit 1
      fi
      out="$(openstack application credential create "${name}" \
        --description "Clutter on this machine (Use this machine's DevStack)" -f json)"
      # openstackclient has used both "id"/"secret" and "ID"/"Secret" as keys.
      if ! printf '%s' "${out}" | python3 -c '
import json, sys
d = json.load(sys.stdin)
c = {"id": d.get("id") or d.get("ID") or "", "secret": d.get("secret") or d.get("Secret") or "", "region": sys.argv[1]}
if not c["id"] or not c["secret"]:
    sys.exit(1)
print(json.dumps(c))
' "${region}"; then
        openstack application credential delete "${name}" || true
        echo "Made the ${name} credential, but could not read its id and secret from the openstack client's output, so it is deleted." >&2
        exit 1
      fi
      ;;
    remove)
      if openstack application credential show "${name}" >/dev/null 2>&1; then
        openstack application credential delete "${name}"
      fi
      ;;
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
      # openstackclient has used both "id"/"secret" and "ID"/"Secret" as keys.
      id="$(printf '%s' "${out}" | python3 -c 'import json, sys; d = json.load(sys.stdin); print(d.get("id") or d.get("ID") or "")')"
      secret="$(printf '%s' "${out}" | python3 -c 'import json, sys; d = json.load(sys.stdin); print(d.get("secret") or d.get("Secret") or "")')"
      if [[ -z "${id}" || -z "${secret}" ]]; then
        echo "Made the ${NAME} credential, but could not read its id and secret from the openstack client's output." >&2
        echo "Run npm run devstack -- credential again to replace it." >&2
        exit 1
      fi
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
      echo "usage: credential.sh check|create|link NAME|remove NAME" >&2
      exit 2
      ;;
  esac
}

main "$@"
exit
