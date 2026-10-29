// pkg/urlpolicy/urlpolicy.go
package urlpolicy

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/linkspire/linkspire/pkg/ssrfguard"
)

const MaxLen = 2048

// Known shorteners used to flag redirect-to-shortener chains (heuristic, not a block).
var KnownShorteners = map[string]struct{}{
	"bit.ly": {}, "tinyurl.com": {}, "t.co": {}, "goo.gl": {}, "ow.ly": {},
	"is.gd": {}, "buff.ly": {}, "rebrand.ly": {}, "short.io": {}, "cutt.ly": {},
	"lnkd.in": {}, "youtu.be": {},
}

// ValidateDestination enforces scheme, length, and SSRF-literal checks at creation time.
// DNS is NOT resolved here (creation must stay fast); workers re-validate with pinning.
func ValidateDestination(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("destination URL is required")
	}
	if len(raw) > MaxLen {
		return "", fmt.Errorf("destination URL exceeds %d bytes", MaxLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if err := ssrfguard.ValidateURLScheme(u.Scheme); err != nil {
		return "", fmt.Errorf("scheme %q is not allowed", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("destination URL must include a host")
	}
	host := u.Hostname()
	if ip := ssrfguard.ParseHostIP(host); ip != nil && ssrfguard.IsBlockedIP(ip) {
		return "", fmt.Errorf("destination host resolves to a blocked address")
	}
	if strings.ContainsAny(raw, "\r\n") {
		return "", fmt.Errorf("destination URL contains control characters")
	}
	// Canonicalize: lowercase host, drop fragment (never sent on redirect anyway).
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String(), nil
}

// HostLooksLikeShortener reports whether host is a known shortener (heuristic).
func HostLooksLikeShortener(host string) bool {
	host = strings.ToLower(host)
	_, ok := KnownShorteners[host]
	if ok {
		return true
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		_, ok = KnownShorteners[h]
		return ok
	}
	return false
}

// HasHomoglyph reports likely IDN / lookalike tricks in the host.
func HasHomoglyph(host string) bool {
	host = strings.ToLower(host)
	if strings.HasPrefix(host, "xn--") || strings.Contains(host, ".xn--") {
		return true
	}
	for _, r := range host {
		if r > 127 {
			return true
		}
		if !utf8.ValidRune(r) {
			return true
		}
	}
	return false
}
