package rss

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// validateFeedURL rejects URLs that we will not fetch: non-http(s) schemes,
// bare hosts, and hostnames that resolve to blocked address ranges. Callers
// should invoke this before handing the URL to gofeed.
func validateFeedURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q; only http and https are allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL missing hostname")
	}
	return nil
}

// isBlockedIP reports whether the given IP is in a range that should never be
// reachable from a feed fetch: loopback, private RFC1918, link-local, cloud
// metadata service addresses, and similar.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip.IsPrivate() { // RFC1918 + IPv6 ULA
		return true
	}
	// AWS/GCP/Azure IMDS addresses are covered by link-local (169.254.169.254,
	// fe80::a9fe:a9fe) but we also explicitly block 100.64.0.0/10 (CGNAT) which
	// Go does not consider private and is sometimes used by metadata services.
	cgnat := &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}
	if cgnat.Contains(ip) {
		return true
	}
	return false
}

// safeDialContext returns a dialer that resolves the target and refuses to
// connect if any resolved address is in a blocked range. Redirect targets go
// through the same dialer, so blocklist applies transitively.
func safeDialContext(resolver *net.Resolver, base *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("dns lookup %q: %w", host, err)
		}
		for _, ip := range ips {
			if isBlockedIP(ip.IP) {
				return nil, fmt.Errorf("address %s resolves to blocked range", host)
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses for %s", host)
		}
		// Dial the first acceptable address explicitly so the resolver result
		// above is the one actually used (prevents TOCTOU via multiple A records
		// returned in a different order at dial time).
		return base.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

// newSafeHTTPClient returns an http.Client suitable for feed fetching that
// refuses connections to blocked IP ranges, follows at most a small number of
// redirects, and enforces a total timeout.
func newSafeHTTPClient(timeout time.Duration) *http.Client {
	base := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext:           safeDialContext(net.DefaultResolver, base),
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return validateFeedURL(req.URL.String())
		},
	}
}
