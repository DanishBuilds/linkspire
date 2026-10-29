// services/redirect/internal/privacy/privacy.go
package privacy

import (
	"crypto/sha256"
	"encoding/binary"
	"net"
	"net/url"
	"strings"

	ua "github.com/mileusna/useragent"
)

// Derived holds the privacy-safe signals extracted from a request. There is deliberately NO
// field for a raw IP or a full user-agent string — those are consumed in memory and dropped.
type Derived struct {
	Country        string
	Region         string
	DeviceClass    string // desktop | mobile | tablet | bot | other
	BrowserFamily  string
	OSFamily       string
	ReferrerDomain string
	IsBot          bool
	VisitorHash    uint64
	Fingerprint    string // anonymized, from truncated IP — for rate limiting only
}

// GeoResolver maps an IP to a coarse location. The default is a stub; a MaxMind-backed
// implementation can be injected (the IP is passed only to derive country/region, then the
// caller discards it).
type GeoResolver interface {
	Lookup(ip net.IP) (country, region string)
}

// StubGeo returns unknown location; used when no GeoIP database is configured.
type StubGeo struct{}

func (StubGeo) Lookup(net.IP) (string, string) { return "XX", "" }

// TruncateIP zeroes host bits for privacy: IPv4 → /24, IPv6 → /48. Returned IP is only ever
// used to derive an anonymized fingerprint / salted visitor hash, never stored raw.
func TruncateIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32))
	}
	return ip.Mask(net.CIDRMask(48, 128))
}

// VisitorHash computes a daily-salted, non-reversible visitor estimate from the IP. Because
// the salt rotates (and is destroyed within 24–48h), the hash cannot correlate a visitor
// across days. The raw IP is NOT part of the output and must be discarded by the caller.
func VisitorHash(ip net.IP, salt string) uint64 {
	if ip == nil {
		return 0
	}
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte{0})
	h.Write(TruncateIP(ip))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

// Fingerprint is an anonymized per-client key for rate limiting, derived from the truncated
// IP (and optionally the browser family). It is not stored and not reversible to a raw IP.
func Fingerprint(ip net.IP, browserFamily string) string {
	h := sha256.New()
	h.Write(TruncateIP(ip))
	h.Write([]byte{0})
	h.Write([]byte(browserFamily))
	sum := h.Sum(nil)
	return encodeHex(sum[:12])
}

const hexDigits = "0123456789abcdef"

func encodeHex(b []byte) string {
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexDigits[v>>4]
		out[i*2+1] = hexDigits[v&0x0f]
	}
	return string(out)
}

// ClassifyUA parses a user-agent string in memory into coarse families, then the caller drops
// the raw UA. Never log or persist the input string.
func ClassifyUA(uaString string) (deviceClass, browserFamily, osFamily string, isBot bool) {
	a := ua.Parse(uaString)
	switch {
	case a.Bot:
		return "bot", famOrOther(a.Name), famOrOther(a.OS), true
	case a.Tablet:
		return "tablet", famOrOther(a.Name), famOrOther(a.OS), false
	case a.Mobile:
		return "mobile", famOrOther(a.Name), famOrOther(a.OS), false
	case a.Desktop:
		return "desktop", famOrOther(a.Name), famOrOther(a.OS), false
	default:
		return "other", famOrOther(a.Name), famOrOther(a.OS), false
	}
}

func famOrOther(s string) string {
	if s == "" {
		return "other"
	}
	return s
}

// ReferrerDomain extracts only the host from a Referer header, stripping path and query (no
// per-page referrer is retained — privacy requirement).
func ReferrerDomain(referer string) string {
	if referer == "" {
		return ""
	}
	u, err := url.Parse(referer)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return strings.ToLower(host)
}

// ClientIP extracts the best-effort client IP from standard proxy headers, used only in memory.
func ClientIP(xff, xRealIP, remoteAddr string) net.IP {
	if xff != "" {
		// first entry is the original client
		first := xff
		if i := strings.IndexByte(xff, ','); i >= 0 {
			first = xff[:i]
		}
		if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
			return ip
		}
	}
	if xRealIP != "" {
		if ip := net.ParseIP(strings.TrimSpace(xRealIP)); ip != nil {
			return ip
		}
	}
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	return net.ParseIP(host)
}
