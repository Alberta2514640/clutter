package netguard

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestBlocked(t *testing.T) {
	for _, ip := range []string{"203.0.113.10", "2001:db8::1", "8.8.8.8"} {
		if Blocked(net.ParseIP(ip), false) {
			t.Errorf("public address %s refused", ip)
		}
	}
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "100.64.0.1", "fd12:3456::1", "::1", "::ffff:10.0.0.5"} {
		if !Blocked(net.ParseIP(ip), false) {
			t.Errorf("private address %s accepted", ip)
		}
	}
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "fd00::1"} {
		if Blocked(net.ParseIP(ip), true) {
			t.Errorf("with private targets allowed, %s refused", ip)
		}
	}
	for _, ip := range []string{"169.254.169.254", "fd00:ec2::254", "0.0.0.0", "0.0.0.1", "fe80::1", "224.0.0.1"} {
		if !Blocked(net.ParseIP(ip), true) {
			t.Errorf("with private targets allowed, %s accepted", ip)
		}
	}
	if !Blocked(nil, true) {
		t.Error("a missing address is accepted")
	}
}

type fakeResolver map[string][]string

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	var out []net.IPAddr
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

func TestCheckHost(t *testing.T) {
	r := fakeResolver{
		"cloud.example.com":    {"203.0.113.10"},
		"inside.example.com":   {"10.0.0.5"},
		"mixed.example.com":    {"203.0.113.10", "169.254.169.254"},
		"empty.example.com":    {},
		"metadata.example.com": {"fd00:ec2::254"},
	}
	for host, wantBlocked := range map[string]bool{
		"cloud.example.com":    false,
		"203.0.113.11":         false,
		"inside.example.com":   true,
		"mixed.example.com":    true,
		"metadata.example.com": true,
		"10.1.2.3":             true,
		"::1":                  true,
	} {
		err := CheckHost(context.Background(), r, host)
		if got := errors.Is(err, ErrBlockedHost); got != wantBlocked || (!wantBlocked && err != nil) {
			t.Errorf("CheckHost(%q) = %v, want blocked=%v", host, err, wantBlocked)
		}
	}
	for _, host := range []string{"missing.example.com", "empty.example.com"} {
		if err := CheckHost(context.Background(), r, host); err == nil {
			t.Errorf("CheckHost(%q) accepted a host with no address", host)
		}
	}
}
