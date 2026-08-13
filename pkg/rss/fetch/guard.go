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

// resolver resolves a hostname to its IP addresses. Satisfied by
// *net.Resolver; unexported because the only production caller is DialContext
// itself — tests substitute a fake via ContextWithResolver/fetchtest to
// exercise DNS-based bypass attempts without a real DNS server. An unexported
// interface can still be implemented and passed in from another package
// (Go's interface satisfaction is structural), which is how fetchtest.
// FakeResolver works.
type resolver interface {
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
// link-local, and unspecified ranges are still blocked. Production code must
// never call this — see fetchtest.AllowLoopback for the intended test-only
// entry point.
func AllowLoopbackForTesting(ctx context.Context) context.Context {
	return context.WithValue(ctx, allowLoopbackContextKey, true)
}

// ContextWithResolver overrides the resolver the guarded dialer uses to look
// up hostnames. It exists for tests that need to exercise a hostname
// resolving to a blocked IP without standing up a real DNS server — see
// fetchtest.WithResolver for the intended test-only entry point.
func ContextWithResolver(ctx context.Context, r resolver) context.Context {
	return context.WithValue(ctx, resolverContextKey, r)
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
		res := resolver(net.DefaultResolver)
		if r, ok := ctx.Value(resolverContextKey).(resolver); ok && r != nil {
			res = r
		}
		resolved, err := res.LookupIPAddr(ctx, host)
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
