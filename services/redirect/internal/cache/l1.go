// services/redirect/internal/cache/l1.go
package cache

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
)

// L1 is the per-pod in-process cache: an LRU with a short TTL. It is the first tier on the
// hot path and absorbs the heavy-hitter links with zero network cost (<1µs lookups).
type L1 struct {
	c *lru.LRU[string, model.Link]
}

// NewL1 builds an L1 cache holding up to size entries, each living at most ttl.
func NewL1(size int, ttl time.Duration) *L1 {
	return &L1{c: lru.NewLRU[string, model.Link](size, nil, ttl)}
}

// Get returns a cached link by key (composed "domainID:code").
func (l *L1) Get(key string) (model.Link, bool) {
	return l.c.Get(key)
}

// Add inserts/refreshes a link.
func (l *L1) Add(key string, v model.Link) {
	l.c.Add(key, v)
}

// Remove evicts a key (used by cache-busting on edit/delete/DSAR).
func (l *L1) Remove(key string) {
	l.c.Remove(key)
}

// Len reports the current entry count.
func (l *L1) Len() int { return l.c.Len() }
