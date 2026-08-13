package fetch

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrBlockedAddress is returned when the guarded dialer refuses to connect to
// a resolved IP because it falls in a loopback, private (RFC1918/RFC4193),
// link-local, or unspecified range. This covers the 169.254.169.254
// cloud-metadata address, which lives in the link-local range.
var ErrBlockedAddress = errors.New("fetch: blocked address (loopback, private, link-local, or unspecified)")

// Resolver resolves a hostname to its IP addresses. Satisfied by
// *net.Resolver; tests may substitute a fake to exercise DNS-based bypass
// attempts without a real DNS server.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type contextKey int

const (
	resolverContextKey contextKey = iota
	allowLoopbackContextKey
)

// AllowLoopbackForTesting returns a context that permits the guarded dialer
// to dial loopback addresses. It exists solely so httptest-based tests (which
// bind their servers to 127.0.0.1) can exercise a real fetch; RFC1918,
// link-local, and unspecified ranges are still blocked. Never call this
// outside a test.
func AllowLoopbackForTesting(ctx context.Context) context.Context {
	return context.WithValue(ctx, allowLoopbackContextKey, true)
}

// ContextWithResolver overrides the Resolver the guarded dialer uses to look
// up hostnames. It exists for tests that need to exercise a hostname
// resolving to a blocked IP without standing up a real DNS server.
func ContextWithResolver(ctx context.Context, r Resolver) context.Context {
	return context.WithValue(ctx, resolverContextKey, r)
}

// FakeResolver is a Resolver returning a fixed set of IPs per hostname, for
// tests exercising DNS-based bypass scenarios.
type FakeResolver map[string][]net.IP

// LookupIPAddr implements Resolver.
func (f FakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	addrs := make([]net.IPAddr, len(ips))
	for i, ip := range ips {
		addrs[i] = net.IPAddr{IP: ip}
	}
	return addrs, nil
}

func isBlockedIP(ip net.IP, allowLoopback bool) bool {
	if allowLoopback && ip.IsLoopback() {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// DialContext is a net.Dialer.DialContext replacement that resolves addr's
// host, rejects the dial if any resolved IP is blocked (see ErrBlockedAddress),
// and only then connects — to the exact IP that was just validated, so DNS
// can't be changed between the check and the connect. Assign it to
// http.Transport.DialContext for any client that fetches caller-supplied
// URLs.
func DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("fetch: split host/port %q: %w", addr, err)
	}

	var ips []net.IP
	if literal := net.ParseIP(host); literal != nil {
		ips = []net.IP{literal}
	} else {
		resolver := Resolver(net.DefaultResolver)
		if r, ok := ctx.Value(resolverContextKey).(Resolver); ok && r != nil {
			resolver = r
		}
		resolved, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("fetch: resolve %q: %w", host, err)
		}
		for _, a := range resolved {
			ips = append(ips, a.IP)
		}
	}

	allowLoopback, _ := ctx.Value(allowLoopbackContextKey).(bool)
	for _, ip := range ips {
		if isBlockedIP(ip, allowLoopback) {
			return nil, fmt.Errorf("%w: %s", ErrBlockedAddress, ip)
		}
	}

	dialer := &net.Dialer{}
	var lastErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("fetch: no addresses resolved for %q", host)
	}
	return nil, lastErr
}
