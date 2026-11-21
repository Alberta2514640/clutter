package generalopenstack

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
)

// firewallGraph is the sample with the firewall's two fields replaced.
func firewallGraph(ingress, source string) connectors.Graph {
	g := sampleGraph()
	g.Blocks["n3"].Fields["ingress"] = ingress
	g.Blocks["n3"].Fields["source_cidr"] = source
	return g
}

// TestFirewallRendersICMPAndIPv6 renders the portable firewall grammar every
// connector reads: icmp has no ports, and is ipv6-icmp for an IPv6 source,
// whose rules carry the IPv6 ethertype.
func TestFirewallRendersICMPAndIPv6(t *testing.T) {
	c := newConnector(nil)
	cases := map[string]struct {
		g    connectors.Graph
		want []string
	}{
		"icmp from IPv4": {
			firewallGraph("tcp/22, icmp", "203.0.113.0/24"),
			[]string{
				`resource "openstack_networking_secgroup_rule_v2" "b_n3_icmp" {` + "\n" +
					"  security_group_id = openstack_networking_secgroup_v2.b_n3.id\n" +
					"  direction         = \"ingress\"\n" +
					"  ethertype         = \"IPv4\"\n" +
					"  protocol          = \"icmp\"\n" +
					"  remote_ip_prefix  = \"203.0.113.0/24\"\n}",
			},
		},
		"tcp and icmp from IPv6": {
			firewallGraph("tcp/443, icmp", "2001:db8::/32"),
			[]string{
				`resource "openstack_networking_secgroup_rule_v2" "b_n3_icmp" {` + "\n" +
					"  security_group_id = openstack_networking_secgroup_v2.b_n3.id\n" +
					"  direction         = \"ingress\"\n" +
					"  ethertype         = \"IPv6\"\n" +
					"  protocol          = \"ipv6-icmp\"\n" +
					"  remote_ip_prefix  = \"2001:db8::/32\"\n}",
				"  ethertype         = \"IPv6\"\n" +
					"  protocol          = \"tcp\"\n" +
					"  port_range_min    = 443\n" +
					"  port_range_max    = 443\n" +
					"  remote_ip_prefix  = \"2001:db8::/32\"",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			files, err := c.Render(tc.g, testDeployment(fixedDeployment))
			if err != nil {
				t.Fatal(err)
			}
			all := string(joined(files))
			for _, w := range tc.want {
				if !strings.Contains(all, w) {
					t.Errorf("missing:\n%s\nin:\n%s", w, all)
				}
			}
		})
	}
}

// TestFirewallRefusesWhatTheGrammarRefuses: the shared grammar refuses a
// source off its network address, and OpenStack keeps its own cap on rules.
func TestFirewallRefusesWhatTheGrammarRefuses(t *testing.T) {
	c := newConnector(nil)
	d := testDeployment(fixedDeployment)
	if ps := c.Validate(firewallGraph("tcp/22", "203.0.113.5/24"), d); !contract.HasProblem(ps, "n3", connectors.CodeInvalidValue) {
		t.Fatalf("a source off its network address: %+v", ps)
	}
	var entries []string
	for p := 1; p <= maxIngressRules+1; p++ {
		entries = append(entries, "tcp/"+strconv.Itoa(p))
	}
	if ps := c.Validate(firewallGraph(strings.Join(entries, ","), "0.0.0.0/0"), d); !contract.HasProblem(ps, "n3", connectors.CodeInvalidValue) {
		t.Fatalf("%d rules: %+v", maxIngressRules+1, ps)
	}
}
