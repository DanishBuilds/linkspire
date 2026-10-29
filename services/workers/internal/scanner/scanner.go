// services/workers/internal/scanner/scanner.go
package scanner

import (
	"bufio"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/linkspire/linkspire/pkg/scan"
	"github.com/linkspire/linkspire/pkg/ssrfguard"
	"github.com/linkspire/linkspire/services/workers/internal/store"
)

// Pipeline runs pluggable scanners and writes the worst verdict.
type Pipeline struct {
	Store    *store.Store
	Scanners []scan.Scanner
	Log      *slog.Logger
}

func (p *Pipeline) Tick(ctx context.Context) {
	links, err := p.Store.LinksNeedingScan(ctx, 50)
	if err != nil {
		p.Log.Error("scan list", "err", err)
		return
	}
	for _, l := range links {
		worst := scan.Clean
		for _, sc := range p.Scanners {
			r, err := sc.Scan(ctx, l.Destination)
			if err != nil {
				p.Log.Warn("scanner error", "scanner", sc.Name(), "err", err)
				continue
			}
			_ = p.Store.SaveScan(ctx, l, r.Scanner, string(r.Verdict), r.Detail)
			worst = scan.Worst(worst, r.Verdict)
		}
		if worst == scan.Clean || worst == scan.Suspicious || worst == scan.Malicious {
			_ = p.Store.SaveScan(ctx, l, "pipeline", string(worst), map[string]any{"final": true})
		}
	}
}

// URLHaus is an async feed scanner. It periodically loads a URL list (CSV) into memory.
type URLHaus struct {
	src string
	mu  sync.RWMutex
	set map[string]struct{}
	log *slog.Logger
}

func NewURLHaus(src string, log *slog.Logger) *URLHaus {
	u := &URLHaus{src: src, set: map[string]struct{}{}, log: log}
	if src != "" {
		go u.refreshLoop()
	}
	return u
}

func (u *URLHaus) Name() string { return "urlhaus" }

func (u *URLHaus) Scan(_ context.Context, destURL string) (scan.Result, error) {
	u.mu.RLock()
	_, hit := u.set[strings.TrimRight(destURL, "/")]
	u.mu.RUnlock()
	if hit {
		return scan.Result{Scanner: u.Name(), Verdict: scan.Malicious, Detail: map[string]any{"feed": "urlhaus"}}, nil
	}
	return scan.Result{Scanner: u.Name(), Verdict: scan.Clean, Detail: map[string]any{"feed": "urlhaus"}}, nil
}

func (u *URLHaus) refreshLoop() {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	u.refresh()
	for range t.C {
		u.refresh()
	}
}

func (u *URLHaus) refresh() {
	client := ssrfguard.NewClient(ssrfguard.DefaultConfig())
	resp, err := client.Get(u.src)
	if err != nil {
		u.log.Warn("urlhaus fetch", "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	set := map[string]struct{}{}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// CSV: id,dateadded,url,... — take the URL column if present.
		parts := strings.Split(line, ",")
		url := line
		if len(parts) >= 3 && strings.HasPrefix(parts[2], "http") {
			url = strings.Trim(parts[2], `"`)
		}
		set[strings.TrimRight(url, "/")] = struct{}{}
	}
	u.mu.Lock()
	u.set = set
	u.mu.Unlock()
}
