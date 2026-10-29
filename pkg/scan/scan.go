// pkg/scan/scan.go
package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/linkspire/linkspire/pkg/urlpolicy"
)

// Verdict mirrors the scan_verdict enum.
type Verdict string

const (
	Pending    Verdict = "pending"
	Clean      Verdict = "clean"
	Suspicious Verdict = "suspicious"
	Malicious  Verdict = "malicious"
)

// Result is one scanner's opinion of a destination.
type Result struct {
	Scanner string         `json:"scanner"`
	Verdict Verdict        `json:"verdict"`
	Detail  map[string]any `json:"detail"`
}

// Scanner is the pluggable interface behind GSB, URLHaus, heuristics, etc.
type Scanner interface {
	Name() string
	Scan(ctx context.Context, destURL string) (Result, error)
}

// Worst returns the most severe of a and b.
func Worst(a, b Verdict) Verdict {
	rank := map[Verdict]int{Clean: 1, Pending: 2, Suspicious: 3, Malicious: 4}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// ---- Heuristics (fast, in-process, <20ms) ----

// brand bait often used in phishing hosts.
var brandKeywords = []string{
	"paypal", "appleid", "apple-id", "icloud", "microsoft", "office365",
	"googlemail", "wellsfargo", "bankofamerica", "chase-secure", "coinbase",
	"binance", "metamask", "whatsapp", "instagram", "facebook", "netflix-billing",
}

// Heuristics is the always-on fast scanner used synchronously at creation.
type Heuristics struct{}

func (Heuristics) Name() string { return "heuristics" }

func (Heuristics) Scan(_ context.Context, destURL string) (Result, error) {
	u, err := url.Parse(destURL)
	if err != nil {
		return Result{Scanner: "heuristics", Verdict: Malicious, Detail: map[string]any{"reason": "unparseable"}}, nil
	}
	host := strings.ToLower(u.Hostname())
	detail := map[string]any{}
	v := Clean

	if urlpolicy.HasHomoglyph(host) {
		v = Worst(v, Suspicious)
		detail["homoglyph"] = true
	}
	if urlpolicy.HostLooksLikeShortener(host) {
		v = Worst(v, Suspicious)
		detail["shortener_chain"] = true
	}
	for _, b := range brandKeywords {
		if strings.Contains(host, b) && !strings.HasSuffix(host, b+".com") && host != b+".com" {
			v = Worst(v, Suspicious)
			detail["brand_bait"] = b
			break
		}
	}
	if strings.Count(host, ".") >= 4 {
		v = Worst(v, Suspicious)
		detail["deep_subdomain"] = true
	}
	return Result{Scanner: "heuristics", Verdict: v, Detail: detail}, nil
}

// ---- Local Safe Browsing hash-prefix DB ----

// HashPrefixDB is a local SHA-256 prefix set (Google Safe Browsing Update API shape:
// 4-byte prefixes of SHA256(canonical URL)). Loaded from a text file of hex prefixes.
type HashPrefixDB struct {
	mu       sync.RWMutex
	prefixes map[string]struct{}
}

func (d *HashPrefixDB) Name() string { return "safe_browsing" }

// LoadFile reads one hex prefix per line. Empty/missing file → empty DB (fail-open to clean).
func (d *HashPrefixDB) LoadFile(path string) error {
	if path == "" {
		d.mu.Lock()
		d.prefixes = map[string]struct{}{}
		d.mu.Unlock()
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			d.mu.Lock()
			d.prefixes = map[string]struct{}{}
			d.mu.Unlock()
			return nil
		}
		return err
	}
	set := map[string]struct{}{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[line] = struct{}{}
	}
	d.mu.Lock()
	d.prefixes = set
	d.mu.Unlock()
	return nil
}

func (d *HashPrefixDB) Scan(_ context.Context, destURL string) (Result, error) {
	sum := sha256.Sum256([]byte(destURL))
	hexSum := hex.EncodeToString(sum[:])
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.prefixes == nil {
		return Result{Scanner: d.Name(), Verdict: Clean, Detail: map[string]any{"db": "empty"}}, nil
	}
	for p := range d.prefixes {
		if strings.HasPrefix(hexSum, p) {
			return Result{Scanner: d.Name(), Verdict: Malicious, Detail: map[string]any{"prefix": p}}, nil
		}
	}
	return Result{Scanner: d.Name(), Verdict: Clean, Detail: map[string]any{"checked": true}}, nil
}

// FastPath runs the synchronous <20ms checks used at link creation: heuristics + local GSB.
func FastPath(ctx context.Context, dest string, gsb *HashPrefixDB) Verdict {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	v := Clean
	if r, err := (Heuristics{}).Scan(ctx, dest); err == nil {
		v = Worst(v, r.Verdict)
	}
	if gsb != nil {
		if r, err := gsb.Scan(ctx, dest); err == nil {
			v = Worst(v, r.Verdict)
		}
	}
	return v
}
