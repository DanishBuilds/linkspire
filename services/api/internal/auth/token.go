// services/api/internal/auth/token.go
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Principal is the authenticated actor. No raw IP / password / full UA.
type Principal struct {
	UserID      int64
	APIKeyID    int64
	WorkspaceID int64
	Role        string
	Scopes      []string
}

func (p Principal) HasRole(min string) bool {
	rank := map[string]int{"viewer": 1, "member": 2, "admin": 3, "owner": 4}
	return rank[p.Role] >= rank[min]
}

func (p Principal) CanWrite() bool  { return p.HasRole("member") }
func (p Principal) CanAdmin() bool  { return p.HasRole("admin") }
func (p Principal) CanOwner() bool  { return p.HasRole("owner") }

type claims struct {
	Sub int64 `json:"sub"`
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
}

// IssueJWT mints an HS256 JWT for a user. Email is intentionally omitted (PII minimization).
func IssueJWT(key []byte, userID int64, ttl time.Duration) (string, error) {
	now := time.Now()
	body, err := json.Marshal(claims{Sub: userID, Iat: now.Unix(), Exp: now.Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + sig, nil
}

// ParseJWT verifies an HS256 JWT and returns the user id.
func ParseJWT(key []byte, token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("malformed token")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(want, got) {
		return 0, errors.New("invalid signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}
	var c claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return 0, err
	}
	if time.Now().Unix() >= c.Exp {
		return 0, errors.New("token expired")
	}
	return c.Sub, nil
}

// NewAPIKey generates lk_live_<hex>. Returns plaintext (shown once), prefix, and SHA-256 hash.
func NewAPIKey() (plaintext, prefix string, hash []byte, err error) {
	var b [24]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", nil, err
	}
	secret := base64.RawURLEncoding.EncodeToString(b[:])
	plaintext = "lk_live_" + secret
	if len(plaintext) >= 12 {
		prefix = plaintext[:12]
	} else {
		prefix = plaintext
	}
	sum := sha256.Sum256([]byte(plaintext))
	return plaintext, prefix, sum[:], nil
}

func HashAPIKey(plaintext string) []byte {
	sum := sha256.Sum256([]byte(plaintext))
	return sum[:]
}

func IsAPIKey(token string) bool {
	return strings.HasPrefix(token, "lk_live_") || strings.HasPrefix(token, "lk_test_")
}

// EncryptAESGCM / DecryptAESGCM envelope-encrypt small secrets (TOTP) at rest.
func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
	k := sha256.Sum256(key)
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptAESGCM(key, blob []byte) ([]byte, error) {
	k := sha256.Sum256(key)
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(blob) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

func ParseWorkspaceHeader(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("missing workspace")
	}
	return strconv.ParseInt(s, 10, 64)
}
