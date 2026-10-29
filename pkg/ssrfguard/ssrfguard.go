// pkg/ssrfguard/ssrfguard.go
// Package ssrfguard provides SSRF-hardened HTTP fetching for the scraper and scanner workers.
// It resolves DNS first, blocks private/loopback/link-local/metadata/IPv6-mapped ranges, pins
// the resolved IP for the actual connection (defeating DNS rebinding), and caps redirects.
package ssrfguard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ErrBlocked indicates a host/IP was rejected by the SSRF policy.
var ErrBlocked = errors.New("ssrfguard: destination blocked by SSRF policy")

// ErrScheme indicates a non-HTTP(S) scheme.
var ErrScheme = errors.New("ssrfguard: only http/https allowed")

// ErrBodyTooLarge is returned when a response exceeds the configured byte ceiling.
var ErrBodyTooLarge = errors.New("ssrfguard: response body exceeds size cap")

// blockedCIDRs are ranges that must never be reachable from a fetcher.
var blockedCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16", // link-local incl. 169.254.169.254 cloud metadata
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::1/128",
		"::/128",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
		"::ffff:0:0/96", // IPv4-mapped IPv6
		"64:ff9b::/96",
		"2001:db8::/32", // documentation
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}()

// IsBlockedIP reports whether an IP falls in any denied range. IPv4-mapped IPv6 is folded
// to the embedded v4 address before the CIDR walk.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Fold IPv4-mapped IPv6 (::ffff:x.x.x.x) to v4 first. On that form IsGlobalUnicast is
	// false (the mapped prefix is not a v6 global unicast), which would block the public internet.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if !ip.IsGlobalUnicast() {
		return true
	}
	v4 := ip.To4() != nil
	for _, n := range blockedCIDRs {
		n4 := len(n.Mask) == net.IPv4len
		if v4 != n4 {
			continue
		}
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ParseHostIP parses a host that is itself an IP, including attacker encodings:
// dotted-decimal, dword (e.g. 2852039166 == 169.254.169.254), octal/hex dotted forms,
// and IPv6 (with or without brackets). Returns nil if host is not an IP literal.
func ParseHostIP(host string) net.IP {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	// Dword IPv4: 2852039166 → 169.254.169.254
	if n, err := strconv.ParseUint(host, 0, 32); err == nil && !strings.Contains(host, ".") {
		return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n)).To4()
	}
	// Dotted with per-octet bases (octal 0251, hex 0xa9, decimal).
	if strings.Count(host, ".") == 3 {
		parts := strings.Split(host, ".")
		var b [4]byte
		ok := true
		for i, p := range parts {
			v, err := strconv.ParseUint(p, 0, 8)
			if err != nil {
				ok = false
				break
			}
			b[i] = byte(v)
		}
		if ok {
			return net.IPv4(b[0], b[1], b[2], b[3]).To4()
		}
	}
	return nil
}

// Resolver abstracts DNS resolution so the guard is testable.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// Config tunes the guarded client.
type Config struct {
	MaxRedirects   int
	DialTimeout    time.Duration
	RequestTimeout time.Duration
	MaxBodyBytes   int64
	Resolver       Resolver
}

// DefaultConfig returns safe defaults (5 redirects, 5s total, 2 MB body).
func DefaultConfig() Config {
	return Config{
		MaxRedirects:   5,
		DialTimeout:    3 * time.Second,
		RequestTimeout: 5 * time.Second,
		MaxBodyBytes:   2 << 20,
		Resolver:       net.DefaultResolver,
	}
}

// NewClient returns an *http.Client whose dialer resolves, validates, and pins IPs, and whose
// redirect policy re-validates every hop and caps the chain.
func NewClient(cfg Config) *http.Client {
	if cfg.MaxRedirects <= 0 {
		cfg.MaxRedirects = 5
	}
	if cfg.Resolver == nil {
		cfg.Resolver = net.DefaultResolver
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 2 << 20
	}

	dialer := &net.Dialer{Timeout: cfg.DialTimeout}

	safeDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolveAndValidate(ctx, cfg.Resolver, host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range ips {
			conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if derr == nil {
				return conn, nil
			}
			lastErr = derr
		}
		if lastErr == nil {
			lastErr = syscall.ECONNREFUSED
		}
		return nil, lastErr
	}

	transport := &http.Transport{
		DialContext:           safeDial,
		DisableKeepAlives:     true,
		MaxIdleConns:          0,
		TLSHandshakeTimeout:   cfg.DialTimeout,
		ResponseHeaderTimeout: cfg.RequestTimeout,
		ForceAttemptHTTP2:     false,
	}

	return &http.Client{
		Timeout:   cfg.RequestTimeout,
		Transport: &bodyCapTransport{base: transport, max: cfg.MaxBodyBytes},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("ssrfguard: stopped after %d redirects", cfg.MaxRedirects)
			}
			if err := ValidateURLScheme(req.URL.Scheme); err != nil {
				return err
			}
			// Re-validate the hop host (IP literal or still-to-be-resolved name).
			if ip := ParseHostIP(req.URL.Hostname()); ip != nil && IsBlockedIP(ip) {
				return fmt.Errorf("%w: redirect hop %s", ErrBlocked, ip)
			}
			return nil
		},
	}
}

func resolveAndValidate(ctx context.Context, r Resolver, host string) ([]net.IP, error) {
	if ip := ParseHostIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return nil, fmt.Errorf("%w: %s", ErrBlocked, ip)
		}
		return []net.IP{ip}, nil
	}
	addrs, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		if IsBlockedIP(a.IP) {
			return nil, fmt.Errorf("%w: %s resolves to blocked %s", ErrBlocked, host, a.IP)
		}
		out = append(out, a.IP)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no usable addresses for %s", ErrBlocked, host)
	}
	return out, nil
}

// ValidateURLScheme rejects non-http(s) schemes (data:, javascript:, file:, gopher:, ...).
func ValidateURLScheme(scheme string) error {
	s := strings.ToLower(scheme)
	if s != "http" && s != "https" {
		return ErrScheme
	}
	return nil
}

// bodyCapTransport wraps a RoundTripper to cap response body size.
type bodyCapTransport struct {
	base http.RoundTripper
	max  int64
}

func (t *bodyCapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &limitedReadCloser{r: resp.Body, n: t.max}
	return resp, nil
}

type limitedReadCloser struct {
	r io.ReadCloser
	n int64
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, ErrBodyTooLarge
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	return n, err
}

func (l *limitedReadCloser) Close() error { return l.r.Close() }
