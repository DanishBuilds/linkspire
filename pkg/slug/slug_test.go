// pkg/slug/slug_test.go
package slug

import "testing"

func TestValidateAlias(t *testing.T) {
	reserved := map[string]struct{}{"admin": {}}
	if _, err := ValidateAlias("ok-slug", reserved); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateAlias("admin", reserved); err == nil {
		t.Fatal("reserved")
	}
	if _, err := ValidateAlias("ab", reserved); err == nil {
		t.Fatal("too short")
	}
	if _, err := ValidateAlias("has spaces", reserved); err == nil {
		t.Fatal("spaces")
	}
	if _, err := ValidateAlias("fuckthis", reserved); err == nil {
		t.Fatal("profanity")
	}
}
