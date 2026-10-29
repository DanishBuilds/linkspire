//go:build security

package security

import (
	"testing"

	"github.com/linkspire/linkspire/pkg/codec"
)

// FuzzDecodeNoPanic feeds arbitrary strings to the Base62 decoder; it must
// never panic, only return an error for invalid input.
func FuzzDecodeNoPanic(f *testing.F) {
	for _, seed := range []string{"", "NoRuDiB", "!!!", "0000000", "zzzzzzz", "../../etc"} {
		f.Add(seed)
	}
	f.Fuzz(func(_ *testing.T, s string) {
		_, _ = codec.Decode(s)
	})
}

// FuzzGeneratorRoundTrip asserts Encode/Decode is a bijection over the code
// space for the Feistel generator used to mint short codes.
func FuzzGeneratorRoundTrip(f *testing.F) {
	g, err := codec.NewGenerator([]byte("0123456789abcdef0123456789abcdef"), 7, 4)
	if err != nil {
		f.Fatalf("new generator: %v", err)
	}
	f.Add(uint64(0))
	f.Add(uint64(123456))
	f.Fuzz(func(t *testing.T, counter uint64) {
		counter %= g.Space()
		code, err := g.Encode(counter)
		if err != nil {
			t.Skipf("encode %d: %v", counter, err)
		}
		got, err := g.Decode(code)
		if err != nil {
			t.Fatalf("decode %q: %v", code, err)
		}
		if got != counter {
			t.Fatalf("round-trip mismatch: %d -> %q -> %d", counter, code, got)
		}
	})
}
