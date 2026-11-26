#!/usr/bin/env bash
# Sets up DevStack's public network inside the Lima VM, now and at every boot.
# devstack.sh pipes this file to bash in the VM, as the VM's default user,
# after create installs DevStack and at every start; it never runs on this
# machine. Arguments: the floating range and its gateway (lib.sh owns both).
#
# stack.sh gives br-ex the gateway address, turns on IPv4 forwarding and adds
# a NAT rule for the floating range, but none of these survive a reboot: br-ex
# comes back down and without its address. This installs a systemd oneshot
# unit that sets all three again after Open vSwitch starts, and runs it now.
# Safe to run again.
set -euo pipefail

UNIT=clutter-devstack-network
SCRIPT="/usr/local/sbin/${UNIT}"

main() {
  local range="${1:-}" gateway="${2:-}" prefix ip='([0-9]{1,3}\.){3}[0-9]{1,3}'
  if [[ ! "${range}" =~ ^${ip}/[0-9]{1,2}$ || ! "${gateway}" =~ ^${ip}$ ]]; then
    echo "usage: network.sh FLOATING_RANGE GATEWAY (for example 172.24.4.0/24 172.24.4.1)" >&2
    exit 2
  fi
  prefix="${range#*/}"
  if [[ ! -f /etc/clutter-devstack ]]; then
    echo "This VM was not made from backend/devserver/devstack/lima.yaml, so its network is left alone." >&2
    exit 1
  fi
  if ! ip link show br-ex >/dev/null 2>&1; then
    echo "DevStack's public network is not set up yet (the VM has no br-ex). npm run devstack -- create installs DevStack."
    exit 0
  fi

  # The values were checked above, so they are safe to write into the script.
  sudo tee "${SCRIPT}" >/dev/null <<EOF
#!/bin/sh
# Written by Clutter's backend/devserver/devstack/network.sh. Run at boot by
# ${UNIT}.service: gives DevStack's br-ex its gateway address, turns on IPv4
# forwarding, and adds the NAT rule stack.sh adds for the floating range.
set -eu
i=0
while ! ip link show br-ex >/dev/null 2>&1; do
  i=\$((i + 1))
  if [ "\${i}" -gt 60 ]; then
    echo "br-ex did not appear within 60 seconds" >&2
    exit 1
  fi
  sleep 1
done
ip link set br-ex up
ip addr replace ${gateway}/${prefix} dev br-ex
sysctl -q -w net.ipv4.ip_forward=1
for dev in \$(ip -4 route show default | awk '{ print \$5 }'); do
  iptables -t nat -C POSTROUTING -o "\${dev}" -s ${range} -j MASQUERADE 2>/dev/null ||
    iptables -t nat -A POSTROUTING -o "\${dev}" -s ${range} -j MASQUERADE
done
EOF
  sudo chmod 755 "${SCRIPT}"

  sudo tee "/etc/systemd/system/${UNIT}.service" >/dev/null <<EOF
[Unit]
Description=DevStack's public network for Clutter (br-ex address, forwarding, NAT)
Wants=network-online.target
After=network-online.target ovs-vswitchd.service openvswitch-switch.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=${SCRIPT}

[Install]
WantedBy=multi-user.target
EOF

  sudo systemctl daemon-reload
  sudo systemctl enable --quiet "${UNIT}.service"
  sudo systemctl restart "${UNIT}.service"
  echo "DevStack's public network is set: br-ex has ${gateway}/${prefix}, forwarding is on, and ${range} has NAT. It is set again at every boot."
}

main "$@"
exit
