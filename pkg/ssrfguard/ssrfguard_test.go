// pkg/ssrfguard/ssrfguard_test.go
package ssrfguard

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"10.1.2.3",
		"172.16.5.4",
		"192.168.1.1",
		"169.254.169.254",
		"100.64.0.1",
		"0.0.0.0",
		"::1",
		"fe80::1",
		"fc00::1",
		"::ffff:169.254.169.254",
		"::ffff:10.0.0.1",
		"255.255.255.255",
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if !IsBlockedIP(ip) {
			t.Errorf("expected %s to be BLOCKED", s)
		}
	}
	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		if IsBlockedIP(ip) {
			t.Errorf("expected %s to be ALLOWED", s)
		}
	}
}

func TestParseHostIPEncodings(t *testing.T) {
	// 169.254.169.254 encodings used in real SSRF bypasses.
	meta := net.ParseIP("169.254.169.254").To4()
	cases := []string{
		"169.254.169.254",
		"2852039166",     // dword
		"0xA9FEA9FE",     // hex dword
		"0251.0376.0251.0376", // octal dotted
	}
	for _, c := range cases {
		got := ParseHostIP(c)
		if got == nil || !got.Equal(meta) {
			t.Errorf("ParseHostIP(%q) = %v, want 169.254.169.254", c, got)
		}
		if !IsBlockedIP(got) {
			t.Errorf("encoded metadata IP %q must be blocked", c)
		}
	}
}

func TestClientBlocksRebindingDNS(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Resolver = fakeResolver{ip: net.ParseIP("169.254.169.254")}
	client := NewClient(cfg)
	_, err := client.Get("http://attacker.example/")
	if err == nil {
		t.Fatal("expected SSRF block when host resolves to metadata IP")
	}
	if !errors.Is(err, ErrBlocked) && !containsSubstr(err.Error(), "blocked") {
		t.Fatalf("want ErrBlocked, got %v", err)
	}
}

func TestClientBlocksIPLiteral(t *testing.T) {
	client := NewClient(DefaultConfig())
	_, err := client.Get("http://127.0.0.1/")
	if err == nil {
		t.Fatal("expected block of loopback literal")
	}
}

func TestClientAllowsPublicResolution(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Resolver = fakeResolver{ip: net.ParseIP("93.184.216.34")}
	client := NewClient(cfg)
	_, err := client.Get("http://public.example/")
	if err != nil && errors.Is(err, ErrBlocked) {
		t.Fatalf("public IP must not be SSRF-blocked, got %v", err)
	}
}

func TestValidateURLScheme(t *testing.T) {
	for _, s := range []string{"data", "javascript", "file", "gopher", "ftp", "vbscript"} {
		if ValidateURLScheme(s) == nil {
			t.Errorf("scheme %q should be rejected", s)
		}
	}
	for _, s := range []string{"http", "https"} {
		if ValidateURLScheme(s) != nil {
			t.Errorf("scheme %q should be allowed", s)
		}
	}
}

type fakeResolver struct{ ip net.IP }

func (f fakeResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return []net.IPAddr{{IP: f.ip}}, nil
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
