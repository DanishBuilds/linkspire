// services/redirect/internal/cache/bloom.go
package cache

import (
	"context"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

// bloomKey composes the filter member for a (domainID, code) pair.
func bloomKey(domainID int64, code string) []byte {
	b := make([]byte, 0, len(code)+20)
	b = appendInt(b, domainID)
	b = append(b, ':')
	b = append(b, code...)
	return b
}

func appendInt(b []byte, n int64) []byte {
	if n == 0 {
		return append(b, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	neg := n < 0
	u := uint64(n)
	if neg {
		u = uint64(-n)
	}
	for u > 0 {
		i--
		tmp[i] = byte('0' + u%10)
		u /= 10
	}
	if neg {
		i--
		tmp[i] = '-'
	}
	return append(b, tmp[i:]...)
}

// LocalBloom is an in-process existence filter (bits-and-blooms). Used in dev / single-tenant
// deployments and in tests. Guarded by an RWMutex: Test is read-heavy on the hot path, Add
// happens only on cache-miss backfill. For internet-scale, prefer RedisBloom (shared, 3.3GB)
// so each pod need not hold the full-corpus filter in memory.
type LocalBloom struct {
	mu sync.RWMutex
	f  *bloom.BloomFilter
}

// NewLocalBloom sizes a filter for n items at false-positive rate p.
func NewLocalBloom(n uint, p float64) *LocalBloom {
	return &LocalBloom{f: bloom.NewWithEstimates(n, p)}
}

// MightExist reports whether the code may exist (false ⇒ definitely absent).
func (b *LocalBloom) MightExist(_ context.Context, domainID int64, code string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.f.Test(bloomKey(domainID, code)), nil
}

// Add records a code as existing.
func (b *LocalBloom) Add(_ context.Context, domainID int64, code string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.f.Add(bloomKey(domainID, code))
	return nil
}
