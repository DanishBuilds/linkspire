// pkg/codec/feistel.go
package codec

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// feistel is a keyed, balanced Feistel network used as a format-preserving permutation over
// Z_a × Z_a. Combined with cycle-walking (see Generator) it yields a bijection over the code
// space, so sequential counter values map to uncorrelated, non-enumerable codes (ADR-003).
//
// The round function is HMAC-SHA256(key, round || right); 4 rounds give a strong permutation
// for our purposes (codes are also gated by scanning/abuse controls, not a secrecy boundary
// for high-value secrets). The key lives in Vault and is injected at startup.
type feistel struct {
	key    []byte
	a      uint64 // half-domain size; full Feistel domain is a*a
	rounds int
}

func newFeistel(key []byte, a uint64, rounds int) *feistel {
	return &feistel{key: key, a: a, rounds: rounds}
}

// f is the round function: a keyed PRF reduced modulo a.
func (fe *feistel) f(round int, r uint64) uint64 {
	var buf [12]byte
	binary.BigEndian.PutUint32(buf[0:4], uint32(round))
	binary.BigEndian.PutUint64(buf[4:12], r)
	mac := hmac.New(sha256.New, fe.key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8]) % fe.a
}

// encrypt permutes m in [0, a*a).
func (fe *feistel) encrypt(m uint64) uint64 {
	l := m / fe.a
	r := m % fe.a
	for i := 0; i < fe.rounds; i++ {
		l, r = r, (l+fe.f(i, r))%fe.a
	}
	return l*fe.a + r
}

// decrypt inverts encrypt over [0, a*a).
func (fe *feistel) decrypt(m uint64) uint64 {
	l := m / fe.a
	r := m % fe.a
	for i := fe.rounds - 1; i >= 0; i-- {
		// invert: previous (l,r) satisfied l_new=r_old, r_new=(l_old+f(i,r_old))%a
		l, r = (r+fe.a-fe.f(i, l)%fe.a)%fe.a, l
	}
	return l*fe.a + r
}

// isqrtCeil returns the smallest integer a such that a*a >= n.
func isqrtCeil(n uint64) uint64 {
	a := uint64(math.Sqrt(float64(n)))
	for a*a < n {
		a++
	}
	// guard against float rounding overshoot
	for a > 0 && (a-1)*(a-1) >= n {
		a--
	}
	return a
}
