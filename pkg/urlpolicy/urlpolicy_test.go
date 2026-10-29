// pkg/urlpolicy/urlpolicy_test.go
package urlpolicy

import "testing"

func TestValidateDestination(t *testing.T) {
	ok, err := ValidateDestination("https://example.com/a")
	if err != nil || ok != "https://example.com/a" {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := ValidateDestination("javascript:alert(1)"); err == nil {
		t.Fatal("javascript must be rejected")
	}
	if _, err := ValidateDestination("http://127.0.0.1/"); err == nil {
		t.Fatal("loopback must be rejected")
	}
	if _, err := ValidateDestination("http://169.254.169.254/latest"); err == nil {
		t.Fatal("metadata IP must be rejected")
	}
}

func TestHasHomoglyph(t *testing.T) {
	if !HasHomoglyph("xn--pple-43d.com") {
		t.Fatal("punycode should flag")
	}
	if HasHomoglyph("example.com") {
		t.Fatal("ascii host should not flag")
	}
}
