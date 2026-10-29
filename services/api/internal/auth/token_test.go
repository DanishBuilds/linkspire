// services/api/internal/auth/token_test.go
package auth

import (
	"testing"
	"time"
)

func TestJWTRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef")
	tok, err := IssueJWT(key, 42, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := ParseJWT(key, tok)
	if err != nil || uid != 42 {
		t.Fatalf("uid=%d err=%v", uid, err)
	}
	if _, err := ParseJWT(key, tok+"x"); err == nil {
		t.Fatal("tamper must fail")
	}
}

func TestAPIKeyPrefixAndHash(t *testing.T) {
	p, prefix, hash, err := NewAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(prefix) < 8 || !IsAPIKey(p) {
		t.Fatalf("prefix=%q key=%q", prefix, p)
	}
	h2 := HashAPIKey(p)
	if string(h2) != string(hash) {
		t.Fatal("hash mismatch")
	}
}

func TestAESGCMRoundTrip(t *testing.T) {
	key := []byte("token-key-token-key")
	ct, err := EncryptAESGCM(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := DecryptAESGCM(key, ct)
	if err != nil || string(pt) != "secret" {
		t.Fatalf("pt=%q err=%v", pt, err)
	}
}

func TestTOTPWindow(t *testing.T) {
	raw, _, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	code := hotp(raw, uint64(now.Unix()/30))
	if !VerifyTOTP(raw, code, now) {
		t.Fatal("expected verify")
	}
	if VerifyTOTP(raw, "000000", now) && code == "000000" {
		// extremely unlikely; ignore
	} else if VerifyTOTP(raw, "111111", now) && code != "111111" {
		t.Fatal("wrong code verified")
	}
}
