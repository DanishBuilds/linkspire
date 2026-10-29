// pkg/slug/slug.go
package slug

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	MinLen = 3
	MaxLen = 64
)

// ValidateAlias checks a vanity slug for charset, length, reserved words, and profanity.
// reserved comes from the reserved_slugs table plus the in-process extra list.
func ValidateAlias(raw string, reserved map[string]struct{}) (string, error) {
	s := strings.TrimSpace(raw)
	if len(s) < MinLen || len(s) > MaxLen {
		return "", fmt.Errorf("alias must be %d–%d characters", MinLen, MaxLen)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("alias may only contain letters, digits, '-' and '_'")
	}
	key := strings.ToLower(s)
	if _, ok := reserved[key]; ok {
		return "", fmt.Errorf("alias %q is reserved", s)
	}
	if isProfane(key) {
		return "", fmt.Errorf("alias is not allowed")
	}
	return s, nil
}

// BuiltinReserved are path segments that would collide with the product surface even if
// missing from the reserved_slugs table.
var BuiltinReserved = []string{
	"v1", "api", "admin", "login", "logout", "signup", "healthz", "readyz",
	"metrics", "robots.txt", "favicon.ico", ".well-known", "qr", "privacy",
}

func isProfane(s string) bool {
	for _, w := range profanity {
		if s == w || strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// Modest, product-surface blocklist. Kept small on purpose — the reserved_slugs table is
// the operational extension point.
var profanity = []string{
	"fuck", "shit", "nigger", "nigga", "faggot", "cunt", "rape", "porn",
}
