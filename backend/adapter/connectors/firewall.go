package connectors

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// ProtocolICMP is the one ingress protocol with no ports. A tcp or udp rule
// holds its protocol as written, "tcp" or "udp".
const ProtocolICMP = "icmp"

// IngressRule is one entry of a firewall block's ingress field. A tcp or udp
// rule allows a port range; an icmp rule allows every ICMP type and has no
// ports (both are 0).
type IngressRule struct {
	Protocol string
	FromPort int
	ToPort   int
}

// ingressEntry is one ingress entry: "tcp/22", "udp/1000-2000" or "icmp".
var ingressEntry = regexp.MustCompile(`^(?:(tcp|udp)/([0-9]{1,5})(?:-([0-9]{1,5}))?|(icmp))$`)

// The messages below are shown to the user as problems.
var (
	errIngress = errors.New("Each allowed entry must look like tcp/22, udp/1000-2000 or icmp, separated by commas, with ports from 1 to 65535 and the lower port first.")
	errCIDR    = errors.New("The address range must look like 10.0.0.0/24 (IPv4) or 2001:db8::/32 (IPv6).")
)

// ParseIngress reads a firewall block's ingress field, such as
// "tcp/22, tcp/8000-8100, icmp". It returns the rules in the order given,
// without repeats; an empty field is no rules. It is the one grammar every
// connector reads, so one diagram passes every connector; a connector adds
// only its cloud's limits (how many rules). The error is safe to show the user.
func ParseIngress(s string) ([]IngressRule, error) {
	var out []IngressRule
	seen := map[IngressRule]bool{}
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		m := ingressEntry.FindStringSubmatch(item)
		if m == nil {
			return nil, errIngress
		}
		r := IngressRule{Protocol: ProtocolICMP}
		if m[4] == "" {
			r.Protocol = m[1]
			r.FromPort, _ = strconv.Atoi(m[2])
			r.ToPort = r.FromPort
			if m[3] != "" {
				r.ToPort, _ = strconv.Atoi(m[3])
			}
			if r.FromPort < 1 || r.ToPort > 65535 || r.FromPort > r.ToPort {
				return nil, errIngress
			}
		}
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out, nil
}

// ParseCIDR reads an address range, IPv4 or IPv6, for a network's cidr or a
// firewall's source_cidr. A range that does not start at its network address
// (10.0.0.5/24) is refused, never changed, so the user sees what is built. A
// connector adds only its cloud's limits (the family, the prefix lengths). The
// error is safe to show the user.
func ParseCIDR(s string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(strings.TrimSpace(s))
	if err != nil || p.Addr().Zone() != "" || p.Addr().Is4In6() {
		return netip.Prefix{}, errCIDR
	}
	if p.Masked() != p {
		return netip.Prefix{}, fmt.Errorf("The address range must start at its network address: use %s.", p.Masked())
	}
	return p, nil
}
