// services/redirect/internal/privacy/salt.go
package privacy

import (
	"context"
	"sync/atomic"
	"time"
)

// SaltProvider supplies the current daily visitor salt used by VisitorHash.
type SaltProvider interface {
	Current() string
}

// StaticSalt is a fixed salt for dev/testing (no rotation).
type StaticSalt string

func (s StaticSalt) Current() string { return string(s) }

// RotatingSalt periodically refreshes the salt from a backing store (e.g. Redis). The actual
// daily rotation + destruction of old salts is performed by the platform's salt-rotation job;
// this type just keeps a hot copy in memory and falls back to a static value if the store is
// unreachable, so the hot path never blocks on salt retrieval.
type RotatingSalt struct {
	fetch    func(context.Context) (string, error)
	fallback string
	val      atomic.Value // string
}

// NewRotatingSalt creates the provider with an initial value equal to fallback.
func NewRotatingSalt(fetch func(context.Context) (string, error), fallback string) *RotatingSalt {
	r := &RotatingSalt{fetch: fetch, fallback: fallback}
	r.val.Store(fallback)
	return r
}

// Current returns the last-known salt.
func (r *RotatingSalt) Current() string {
	if v, ok := r.val.Load().(string); ok && v != "" {
		return v
	}
	return r.fallback
}

// Start launches a refresh loop until ctx is cancelled. It refreshes immediately, then every
// interval.
func (r *RotatingSalt) Start(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		r.refresh(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				r.refresh(ctx)
			}
		}
	}()
}

func (r *RotatingSalt) refresh(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if v, err := r.fetch(cctx); err == nil && v != "" {
		r.val.Store(v)
	}
}
