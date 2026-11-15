// Package netguard decides which network addresses B may connect to on a
// connection's behalf: the one owner of that list, for B API's own requests
// (the OpenStack connector's identity check) and for the run task, which
// checks a run's hosts before Terraform starts.
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// metadataIPv6 is the instance metadata address AWS gives inside a VPC. It is
// a unique local address, not link-local, so it is named here.
var metadataIPv6 = net.ParseIP("fd00:ec2::254")

var (
	// thisNetwork (0.0.0.0/8) reaches the local host on some systems.
	thisNetwork = mustCIDR("0.0.0.0/8")
	// sharedAddressSpace (100.64.0.0/10) is carrier-grade NAT space, used
	// inside providers' and overlay networks, never by a public cloud.
	sharedAddressSpace = mustCIDR("100.64.0.0/10")
)

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// Blocked reports whether B refuses to connect to ip. Link-local, metadata,
// unspecified and multicast addresses are always refused, so a connection
// cannot point B at a cloud metadata service. Loopback, private (including
// IPv6 unique local) and shared addresses are refused too, so a connection
// cannot probe the network B runs in, unless allowPrivate is set for a
// laptop's DevStack (A2 only).
func Blocked(ip net.IP, allowPrivate bool) bool {
	if ip == nil || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.Equal(metadataIPv6) || thisNetwork.Contains(ip) {
		return true
	}
	if allowPrivate {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || sharedAddressSpace.Contains(ip)
}

// ErrBlockedHost is wrapped by CheckHost when a host resolves to an address
// Blocked refuses.
var ErrBlockedHost = errors.New("the host resolves to an address Clutter does not connect to")

// Resolver looks up a host's addresses; *net.Resolver is one.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// CheckHost resolves host (a name or a literal address) and refuses it when
// any address it resolves to is blocked, private ones included. The answer
// holds only at the moment of the lookup: a name whose DNS changes later can
// still lead elsewhere, so the network around the caller must refuse those
// addresses too.
func CheckHost(ctx context.Context, r Resolver, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if Blocked(ip, false) {
			return ErrBlockedHost
		}
		return nil
	}
	addrs, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("look up the host: %w", err)
	}
	if len(addrs) == 0 {
		return errors.New("the host has no address")
	}
	for _, a := range addrs {
		if Blocked(a.IP, false) {
			return ErrBlockedHost
		}
	}
	return nil
}
