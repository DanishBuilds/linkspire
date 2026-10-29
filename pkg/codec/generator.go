// pkg/codec/generator.go
package codec

import (
	"errors"
	"fmt"
)

// DefaultWidth is the default short-code length (Base62). 62^7 ≈ 3.52e12 codes per domain.
const DefaultWidth = 7

// DefaultRounds is the Feistel round count. 4 rounds is the standard balanced-Feistel choice
// for a strong format-preserving permutation.
const DefaultRounds = 4

// ErrSpaceExhausted is returned when a counter value is outside the code space.
var ErrSpaceExhausted = errors.New("codec: counter exceeds code space")

// Generator turns a monotonically-increasing internal counter (leased in blocks via a HiLo
// allocator, see Phase 4) into a non-sequential, non-enumerable Base62 code of fixed width.
//
// It composes a keyed Feistel permutation with cycle-walking so the map
//
//	counter ∈ [0, N) → code ∈ [0, N)
//
// is a bijection (collision-free by construction) that scrambles ordering (non-enumerable).
type Generator struct {
	width int
	n     uint64 // code space = base^width
	fe    *feistel
}

// NewGenerator builds a Generator. key must be non-empty (sourced from Vault). width is the
// number of Base62 characters (use DefaultWidth). rounds should be DefaultRounds.
func NewGenerator(key []byte, width, rounds int) (*Generator, error) {
	if len(key) == 0 {
		return nil, errors.New("codec: empty feistel key")
	}
	if width < 1 || width > 10 {
		return nil, fmt.Errorf("codec: width %d out of range [1,10]", width)
	}
	if rounds < 2 {
		return nil, errors.New("codec: rounds must be >= 2")
	}
	var n uint64 = 1
	for i := 0; i < width; i++ {
		n *= base
	}
	a := isqrtCeil(n)
	return &Generator{width: width, n: n, fe: newFeistel(key, a, rounds)}, nil
}

// Width returns the fixed code width.
func (g *Generator) Width() int { return g.width }

// Space returns the total number of codes (N = base^width).
func (g *Generator) Space() uint64 { return g.n }

// permute applies the Feistel permutation with cycle-walking to stay within [0, n).
func (g *Generator) permute(x uint64) uint64 {
	y := g.fe.encrypt(x)
	for y >= g.n {
		y = g.fe.encrypt(y)
	}
	return y
}

// unpermute inverts permute.
func (g *Generator) unpermute(y uint64) uint64 {
	x := g.fe.decrypt(y)
	for x >= g.n {
		x = g.fe.decrypt(x)
	}
	return x
}

// Encode maps an internal counter to its short code.
func (g *Generator) Encode(counter uint64) (string, error) {
	if counter >= g.n {
		return "", ErrSpaceExhausted
	}
	return EncodeFixed(g.permute(counter), g.width), nil
}

// Decode maps a short code back to its internal counter. Used for validation and fuzzing;
// the redirect hot path never decodes (it looks up the stored code directly).
func (g *Generator) Decode(code string) (uint64, error) {
	if len(code) != g.width {
		return 0, fmt.Errorf("codec: code length %d != width %d", len(code), g.width)
	}
	v, err := Decode(code)
	if err != nil {
		return 0, err
	}
	if v >= g.n {
		return 0, ErrSpaceExhausted
	}
	return g.unpermute(v), nil
}
