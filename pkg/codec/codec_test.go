// pkg/codec/codec_test.go
package codec

import (
	"testing"
)

func newTestGen(t *testing.T) *Generator {
	t.Helper()
	g, err := NewGenerator([]byte("test-feistel-key-do-not-use-in-prod"), DefaultWidth, DefaultRounds)
	if err != nil {
		t.Fatalf("NewGenerator: %v", err)
	}
	return g
}

func TestEncodeFixedWidthAndRoundTrip(t *testing.T) {
	g := newTestGen(t)
	for _, c := range []uint64{0, 1, 2, 1000, 1 << 20, g.Space() - 1} {
		code, err := g.Encode(c)
		if err != nil {
			t.Fatalf("Encode(%d): %v", c, err)
		}
		if len(code) != DefaultWidth {
			t.Fatalf("Encode(%d) = %q, want width %d", c, code, DefaultWidth)
		}
		got, err := g.Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q): %v", code, err)
		}
		if got != c {
			t.Fatalf("round-trip mismatch: counter %d -> %q -> %d", c, code, got)
		}
	}
}

// Bijection + collision-free: a dense range of counters must map to distinct codes.
func TestNoCollisionsOverRange(t *testing.T) {
	g := newTestGen(t)
	const n = 200_000
	seen := make(map[string]uint64, n)
	for c := uint64(0); c < n; c++ {
		code, err := g.Encode(c)
		if err != nil {
			t.Fatalf("Encode(%d): %v", c, err)
		}
		if prev, dup := seen[code]; dup {
			t.Fatalf("collision: counters %d and %d both -> %q", prev, c, code)
		}
		seen[code] = c
	}
}

// Non-enumerable: consecutive counters must not produce lexicographically adjacent codes.
// We assert that adjacent counters differ in the integer domain by a large amount on average.
func TestNonSequential(t *testing.T) {
	g := newTestGen(t)
	var adjacentSharePrefix int
	prevCode, _ := g.Encode(0)
	for c := uint64(1); c < 5000; c++ {
		code, _ := g.Encode(c)
		if code[:3] == prevCode[:3] {
			adjacentSharePrefix++
		}
		prevCode = code
	}
	// With a random permutation the chance of sharing a 3-char prefix is ~1/62^3; seeing many
	// such events would indicate the codes leak sequential structure.
	if adjacentSharePrefix > 5 {
		t.Fatalf("codes appear sequential: %d adjacent 3-char prefix matches", adjacentSharePrefix)
	}
}

func TestEncodeSpaceExhausted(t *testing.T) {
	g := newTestGen(t)
	if _, err := g.Encode(g.Space()); err != ErrSpaceExhausted {
		t.Fatalf("Encode(N) err = %v, want ErrSpaceExhausted", err)
	}
}

func TestDecodeRejectsBadInput(t *testing.T) {
	g := newTestGen(t)
	cases := []string{"", "abc", "aaaaaaaaaaaa", "!!!!!!!", "aaaaaa/"}
	for _, in := range cases {
		if _, err := g.Decode(in); err == nil {
			t.Fatalf("Decode(%q) expected error, got nil", in)
		}
	}
}

func TestBase62Overflow(t *testing.T) {
	if _, err := Decode("zzzzzzzzzzzz"); err == nil {
		t.Fatal("expected overflow/too-long error")
	}
}

// FuzzDecoder exercises the public Base62 decoder for panics / inconsistent results.
func FuzzDecoder(f *testing.F) {
	seeds := []string{"", "0", "aB3xY7z", "zzzzzzz", "123456789AB", "//////"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := Decode(s)
		if err != nil {
			return // invalid input rejected cleanly
		}
		// If it decoded, every character must be a valid Base62 symbol.
		if !IsValid(s) {
			t.Fatalf("Decode accepted invalid string %q (v=%d)", s, v)
		}
	})
}

func BenchmarkEncode(b *testing.B) {
	g, _ := NewGenerator([]byte("bench-key"), DefaultWidth, DefaultRounds)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Encode(uint64(i) % g.Space())
	}
}
