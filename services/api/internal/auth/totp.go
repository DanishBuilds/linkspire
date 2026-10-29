// services/api/internal/auth/totp.go
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// NewTOTPSecret returns a 20-byte secret and its base32 encoding (otpauth).
func NewTOTPSecret() (raw []byte, b32 string, err error) {
	raw = make([]byte, 20)
	if _, err = rand.Read(raw); err != nil {
		return nil, "", err
	}
	b32 = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return raw, b32, nil
}

func OTPAuthURL(email, secretB32, issuer string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, email, secretB32, issuer)
}

func hotp(secret []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", bin%1000000)
}

// VerifyTOTP accepts the current 30s window ±1.
func VerifyTOTP(secret []byte, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	ctr := uint64(now.Unix() / 30)
	for _, w := range []int64{-1, 0, 1} {
		if hmac.Equal([]byte(hotp(secret, uint64(int64(ctr)+w))), []byte(code)) {
			return true
		}
	}
	return false
}
