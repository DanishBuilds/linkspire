// pkg/argon2id/argon2id_test.go
package argon2id

import "testing"

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !stringsHasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("unexpected PHC prefix: %s", h[:min(40, len(h))])
	}
	if !Verify("correct horse battery staple", h) {
		t.Fatal("expected match")
	}
	if Verify("wrong password", h) {
		t.Fatal("expected mismatch")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	if Verify("x", "") || Verify("x", "not-a-hash") || Verify("x", "$argon2i$v=19$m=1,t=1,p=1$YQ$YQ") {
		t.Fatal("garbage hashes must not verify")
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
