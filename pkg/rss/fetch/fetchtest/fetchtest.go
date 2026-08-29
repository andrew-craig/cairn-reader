// Package fetchtest provides test-only helpers for exercising pkg/rss/fetch's
// SSRF guard. Nothing here belongs in production code — the name mirrors the
// standard-library convention (net/http/httptest, testing/fstest) of a
// same-tree, unmistakably-test-scoped package, so the loopback bypass and
// fake-DNS helpers a guard test needs don't have to live in fetch's own
// importable surface.
package fetchtest

import (
	"context"
	"net"

	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch"
)

// AllowLoopback returns a context that permits the guarded dialer to dial
// loopback addresses. httptest servers bind to 127.0.0.1; RFC1918,
// link-local, and unspecified ranges remain blocked.
func AllowLoopback(ctx context.Context) context.Context {
	return fetch.AllowLoopbackForTesting(ctx)
}

// FakeResolver returns a fixed set of IPs per hostname, for tests exercising
// DNS-based bypass scenarios without a real DNS server.
type FakeResolver map[string][]net.IP

// LookupIPAddr implements the resolver interface fetch's guarded dialer
// expects.
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

// WithResolver overrides the resolver the guarded dialer uses to look up
// hostnames with r.
func WithResolver(ctx context.Context, r FakeResolver) context.Context {
	return fetch.ContextWithResolver(ctx, r)
}
