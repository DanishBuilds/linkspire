//go:build security

// Package security holds Phase 6 security tests. They are tagged `security` so
// they never run in the default build; execute with:
//
//	go test -tags security ./test/security/...
package security

import (
	"net"
	"testing"

	"github.com/linkspire/linkspire/pkg/ssrfguard"
)

// TestSSRF_BlockedIPs asserts the IP policy blocks loopback, link-local
// (cloud metadata), private, unspecified, and IPv4-mapped-IPv6 ranges.
func TestSSRF_BlockedIPs(t *testing.T) {
	blocked := []string{
		"169.254.169.254",  // cloud metadata
		"127.0.0.1", "::1", // loopback
		"10.0.0.1", "192.168.1.1", "172.16.0.1", // RFC1918
		"0.0.0.0", "::", // unspecified
		"::ffff:169.254.169.254", // IPv4-mapped metadata
		"fd00::1",                // unique local
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("could not parse %q", s)
		}
		if !ssrfguard.IsBlockedIP(ip) {
			t.Errorf("expected %s to be blocked", s)
		}
	}

	allowed := []string{"93.184.216.34", "1.1.1.1", "2606:4700:4700::1111"}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("could not parse %q", s)
		}
		if ssrfguard.IsBlockedIP(ip) {
			t.Errorf("expected %s to be allowed", s)
		}
	}
}

// TestSSRF_SchemeDeny asserts only http/https are permitted.
func TestSSRF_SchemeDeny(t *testing.T) {
	for _, s := range []string{"ftp", "file", "gopher", "javascript", "data", "dict"} {
		if ssrfguard.ValidateURLScheme(s) == nil {
			t.Errorf("scheme %q should be rejected", s)
		}
	}
	for _, s := range []string{"http", "https"} {
		if ssrfguard.ValidateURLScheme(s) != nil {
			t.Errorf("scheme %q should be allowed", s)
		}
	}
}
