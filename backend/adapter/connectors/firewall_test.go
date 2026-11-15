package connectors

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseIngress(t *testing.T) {
	got, err := ParseIngress(" tcp/22, udp/1000-2000,icmp, tcp/22 ,, tcp/80-80")
	if err != nil {
		t.Fatal(err)
	}
	want := []IngressRule{
		{Protocol: ProtocolTCP, FromPort: 22, ToPort: 22},
		{Protocol: ProtocolUDP, FromPort: 1000, ToPort: 2000},
		{Protocol: ProtocolICMP},
		{Protocol: ProtocolTCP, FromPort: 80, ToPort: 80},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if rules, err := ParseIngress("  "); err != nil || len(rules) != 0 {
		t.Fatalf("an empty field: %+v %v", rules, err)
	}
	for _, bad := range []string{"tcp/0", "tcp/70000", "udp/90-80", "icmp/8", "sctp/22", "tcp/+22", "TCP/22", `tcp/22"`, "tcp/22; rm -rf /", "tcp"} {
		if _, err := ParseIngress(bad); err == nil {
			t.Errorf("%q is accepted", bad)
		}
	}
}

func TestParseCIDR(t *testing.T) {
	for in, want := range map[string]string{
		"0.0.0.0/0":         "0.0.0.0/0",
		" 10.0.0.0/24 ":     "10.0.0.0/24",
		"2001:db8::/32":     "2001:db8::/32",
		"2001:0db8:0::/48":  "2001:db8::/48",
		"203.0.113.7/32":    "203.0.113.7/32",
		"::/0":              "::/0",
		"fd00:1:2:3::/64":   "fd00:1:2:3::/64",
		"198.51.100.128/25": "198.51.100.128/25",
	} {
		p, err := ParseCIDR(in)
		if err != nil || p.String() != want {
			t.Errorf("%q: %v %v", in, p, err)
		}
	}
	for _, bad := range []string{"anywhere", "10.0.0.0", "10.0.0.0/33", "${x}", "0.0.0.0/0\n}", "::ffff:10.0.0.0/104"} {
		if _, err := ParseCIDR(bad); err == nil {
			t.Errorf("%q is accepted", bad)
		}
	}
	// A range off its network address is refused, never changed.
	_, err := ParseCIDR("10.0.0.5/24")
	if err == nil || !strings.Contains(err.Error(), "10.0.0.0/24") {
		t.Fatalf("10.0.0.5/24: %v", err)
	}
	if _, err := ParseCIDR("2001:db8::1/32"); err == nil {
		t.Fatal("2001:db8::1/32 is accepted")
	}
}
