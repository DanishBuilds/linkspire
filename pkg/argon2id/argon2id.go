// pkg/argon2id/argon2id.go
// Package argon2id hashes and verifies secrets using Argon2id in PHC string format.
//
// Tuned parameters (documented, used for both account passwords and link passwords):
//
//	memory      = 65536 KiB (64 MiB)
//	iterations  = 3
//	parallelism = 2
//	salt length = 16 bytes
//	key length  = 32 bytes
//
// These take ~50–80 ms on a modern core — acceptable for the management/unlock paths,
// never used on the redirect GET hot path except after an explicit POST unlock.
package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	MemoryKiB  = 65536
	TimeCost   = 3
	Threads    = 2
	SaltLen    = 16
	KeyLen     = 32
	phcVersion = argon2.Version // 19
)

// Hash returns a PHC-encoded Argon2id string of password. Never log the input or output
// in application logs — the output is stored, the input is discarded.
func Hash(password string) (string, error) {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := argon2.IDKey([]byte(password), salt, TimeCost, MemoryKiB, Threads, KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		phcVersion, MemoryKiB, TimeCost, Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	), nil
}

// Verify reports whether password matches a PHC-encoded Argon2id hash. Comparison is
// constant-time on the derived key. Malformed hashes return false (never panic).
func Verify(password, phc string) bool {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != phcVersion {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
