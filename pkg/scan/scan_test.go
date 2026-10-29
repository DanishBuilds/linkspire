// pkg/scan/scan_test.go
package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHeuristicsFlagsHomoglyphAndShortener(t *testing.T) {
	h := Heuristics{}
	r, _ := h.Scan(context.Background(), "https://xn--pple-43d.com/login")
	if r.Verdict != Suspicious {
		t.Fatalf("punycode host: got %s", r.Verdict)
	}
	r, _ = h.Scan(context.Background(), "https://bit.ly/abc")
	if r.Verdict != Suspicious {
		t.Fatalf("shortener: got %s", r.Verdict)
	}
	r, _ = h.Scan(context.Background(), "https://example.com/ok")
	if r.Verdict != Clean {
		t.Fatalf("benign: got %s", r.Verdict)
	}
}

func TestHashPrefixDBHit(t *testing.T) {
	dest := "https://evil.example/phish"
	sum := sha256.Sum256([]byte(dest))
	prefix := hex.EncodeToString(sum[:4])
	dir := t.TempDir()
	path := filepath.Join(dir, "gsb.txt")
	if err := os.WriteFile(path, []byte(prefix+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var db HashPrefixDB
	if err := db.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	r, _ := db.Scan(context.Background(), dest)
	if r.Verdict != Malicious {
		t.Fatalf("expected malicious, got %s", r.Verdict)
	}
	r, _ = db.Scan(context.Background(), "https://example.com")
	if r.Verdict != Clean {
		t.Fatalf("unlisted URL must be clean, got %s", r.Verdict)
	}
}

func TestFastPathTimeoutBudget(t *testing.T) {
	v := FastPath(context.Background(), "https://example.com", nil)
	if v != Clean {
		t.Fatalf("got %s", v)
	}
}
