// pkg/idgen/hilo.go
package idgen

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/linkspire/linkspire/pkg/codec"
)

const defaultBlock = 10_000

// Allocator leases blocks of integers from hilo_counters and feeds them through a
// keyed Feistel generator to produce collision-free, non-enumerable short codes (ADR-003).
type Allocator struct {
	pool  *pgxpool.Pool
	gen   *codec.Generator
	block uint64

	mu   sync.Mutex
	next uint64
	end  uint64
}

// NewAllocator builds a HiLo allocator. block is the lease size (10k default).
func NewAllocator(pool *pgxpool.Pool, gen *codec.Generator, block uint64) *Allocator {
	if block == 0 {
		block = defaultBlock
	}
	return &Allocator{pool: pool, gen: gen, block: block}
}

// NextCode returns the next short code, leasing a new block from the database when needed.
func (a *Allocator) NextCode(ctx context.Context) (string, error) {
	n, err := a.nextCounter(ctx)
	if err != nil {
		return "", err
	}
	return a.gen.Encode(n)
}

func (a *Allocator) nextCounter(ctx context.Context) (uint64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.next >= a.end {
		var hi uint64
		err := a.pool.QueryRow(ctx, `
			UPDATE hilo_counters SET next_hi = next_hi + $1, updated_at = now()
			WHERE name = 'codes'
			RETURNING next_hi
		`, a.block).Scan(&hi)
		if err != nil {
			return 0, fmt.Errorf("idgen: lease block: %w", err)
		}
		a.end = hi
		a.next = hi - a.block
	}
	n := a.next
	a.next++
	return n, nil
}
