//go:build security

package security

import (
	"net/http"
	"os"
	"testing"
)

// TestRateLimitAndSpoofResistance is a best-effort check against a running
// redirect service. It fires a burst at a configurable path and verifies that
// (a) the server never 5xxs under the burst, and (b) spoofing X-Forwarded-For
// does not grant a larger allowance than a single client (no trivial bypass).
//
//	REDIRECT_URL=http://localhost:8080 RL_PATH=/does-not-exist \
//	  go test -tags security -run TestRateLimit ./test/security/...
func TestRateLimitAndSpoofResistance(t *testing.T) {
	base := os.Getenv("REDIRECT_URL")
	if base == "" {
		t.Skip("set REDIRECT_URL to run the rate-limit test")
	}
	path := os.Getenv("RL_PATH")
	if path == "" {
		path = "/"
	}
	client := &http.Client{}

	burst := func(spoof bool) (ok, limited, server int) {
		for i := 0; i < 200; i++ {
			req, _ := http.NewRequest(http.MethodGet, base+path, nil)
			if spoof {
				req.Header.Set("X-Forwarded-For", randIP(i))
			}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			switch {
			case resp.StatusCode == http.StatusTooManyRequests:
				limited++
			case resp.StatusCode >= 500:
				server++
			default:
				ok++
			}
			resp.Body.Close()
		}
		return
	}

	_, limitedSame, server := burst(false)
	if server > 0 {
		t.Errorf("redirect returned %d server errors under burst", server)
	}
	_, limitedSpoof, _ := burst(true)

	// If a limit is enforced, spoofing XFF must not clearly evade it.
	if limitedSame > 0 && limitedSpoof == 0 {
		t.Errorf("possible rate-limit bypass: %d limited without spoofing, 0 limited when spoofing X-Forwarded-For", limitedSame)
	}
	t.Logf("limited(same-client)=%d limited(spoofed-xff)=%d", limitedSame, limitedSpoof)
}

func randIP(i int) string {
	return "203.0.113." + itoa(i%254+1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [3]byte
	p := len(b)
	for n > 0 {
		p--
		b[p] = byte('0' + n%10)
		n /= 10
	}
	return string(b[p:])
}
