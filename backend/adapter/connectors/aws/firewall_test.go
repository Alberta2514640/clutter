package aws

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// firewallGraph is the full graph with the firewall's two fields replaced.
func firewallGraph(ingress, source string) connectors.Graph {
	g := fullGraph()
	g.Blocks["fw"].Fields["ingress"] = ingress
	g.Blocks["fw"].Fields["source_cidr"] = source
	return g
}

// TestFirewallRendersICMPAndIPv6 renders the portable firewall grammar every
// connector reads: icmp allows every ICMP type (-1), as icmpv6 for an IPv6
// source, and an IPv6 source goes in cidr_ipv6.
func TestFirewallRendersICMPAndIPv6(t *testing.T) {
	c, _, _ := newTestConnector(t)
	cases := map[string]struct {
		g        connectors.Graph
		want     []string
		mustNots []string
	}{
		"icmp from IPv4": {
			firewallGraph("tcp/22, icmp", "203.0.113.0/24"),
			[]string{
				"ip_protocol       = \"icmp\"\n  from_port         = -1\n  to_port           = -1\n  cidr_ipv4         = \"203.0.113.0/24\"",
				"ip_protocol       = \"tcp\"\n  from_port         = 22\n  to_port           = 22\n  cidr_ipv4         = \"203.0.113.0/24\"",
			},
			[]string{"cidr_ipv6", "icmpv6"},
		},
		"tcp and icmp from IPv6": {
			firewallGraph("tcp/443, udp/1000-2000, icmp", "2001:db8::/32"),
			[]string{
				"ip_protocol       = \"icmpv6\"\n  from_port         = -1\n  to_port           = -1\n  cidr_ipv6         = \"2001:db8::/32\"",
				"ip_protocol       = \"tcp\"\n  from_port         = 443\n  to_port           = 443\n  cidr_ipv6         = \"2001:db8::/32\"",
				"ip_protocol       = \"udp\"\n  from_port         = 1000\n  to_port           = 2000\n  cidr_ipv6         = \"2001:db8::/32\"",
			},
			[]string{"cidr_ipv4         = \"2001", "\"icmp\""},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			all := joined(renderFor(t, c, tc.g, testDeployment("ca-central-1")))
			for _, w := range tc.want {
				if !strings.Contains(all, w) {
					t.Errorf("missing:\n%s", w)
				}
			}
			for _, n := range tc.mustNots {
				if strings.Contains(all, n) {
					t.Errorf("holds %q", n)
				}
			}
		})
	}
}

// TestFirewallRuleLimit: AWS keeps its own cap on rules, over the shared
// grammar.
func TestFirewallRuleLimit(t *testing.T) {
	c, _, _ := newTestConnector(t)
	var entries []string
	for p := 1; p <= maxIngressRules+1; p++ {
		entries = append(entries, "tcp/"+strconv.Itoa(p))
	}
	g := firewallGraph(strings.Join(entries, ","), "0.0.0.0/0")
	ps := c.Validate(g, testDeployment("ca-central-1"))
	if len(ps) != 1 || ps[0].Field != fieldIngress || ps[0].Code != connectors.CodeInvalidValue {
		t.Fatalf("%d rules: %+v", maxIngressRules+1, ps)
	}
}
