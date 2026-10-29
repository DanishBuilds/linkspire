// pkg/codec/base62.go
package codec

import (
	"errors"
	"strings"
)

// Base62 alphabet. Order is fixed and MUST NOT change once codes are issued, otherwise
// previously generated short codes would decode to different integers.
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const base = 62

// ErrInvalidChar is returned when a string contains a non-Base62 character.
var ErrInvalidChar = errors.New("codec: invalid base62 character")

// ErrTooLong is returned when a string would overflow a uint64 on decode.
var ErrTooLong = errors.New("codec: input too long to decode into uint64")

// decodeTable maps a byte to its Base62 value, or -1 if not part of the alphabet.
var decodeTable [256]int8

func init() {
	for i := range decodeTable {
		decodeTable[i] = -1
	}
	for i := 0; i < len(alphabet); i++ {
		decodeTable[alphabet[i]] = int8(i)
	}
}

// EncodeFixed encodes n into a Base62 string of exactly width characters, left-padded with
// the zero symbol. It panics if n does not fit in width digits (a programming error; callers
// generate n from a bounded domain).
func EncodeFixed(n uint64, width int) string {
	buf := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		buf[i] = alphabet[n%base]
		n /= base
	}
	if n != 0 {
		panic("codec: value does not fit in requested width")
	}
	return string(buf)
}

// Decode parses a Base62 string into a uint64. It rejects unknown characters and inputs that
// would overflow uint64. Leading zero symbols are allowed (fixed-width codes).
func Decode(s string) (uint64, error) {
	if s == "" {
		return 0, ErrInvalidChar
	}
	// 11 Base62 digits is the max that always fits in uint64 (62^11 > 2^64 > 62^10).
	if len(s) > 11 {
		return 0, ErrTooLong
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		v := decodeTable[s[i]]
		if v < 0 {
			return 0, ErrInvalidChar
		}
		hi := n
		n = n*base + uint64(v)
		if n < hi { // overflow wrapped around
			return 0, ErrTooLong
		}
	}
	return n, nil
}

// IsValid reports whether s consists solely of Base62 characters.
func IsValid(s string) bool {
	if s == "" {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool {
		return r > 255 || decodeTable[byte(r)] < 0
	}) == -1
}
