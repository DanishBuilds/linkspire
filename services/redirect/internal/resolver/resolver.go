// services/redirect/internal/resolver/resolver.go
package resolver

import (
	"context"
	"errors"
	"math/rand/v2"
	"strconv"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
	"github.com/linkspire/linkspire/services/redirect/internal/observability"
)

// ErrNotFound indicates the code does not resolve to a link (after all tiers + negative cache).
var ErrNotFound = errors.New("resolver: link not found")

// Source identifies which tier satisfied a lookup (for metrics / debugging).
type Source string

const (
	SourceL1       Source = "l1"
	SourceRedis    Source = "redis"
	SourcePostgres Source = "pg"
)

// L1Cache is the in-process tier.
type L1Cache interface {
	Get(key string) (model.Link, bool)
	Add(key string, v model.Link)
}

// ExistenceFilter is the Bloom tier (local or RedisBloom). MightExist=false ⇒ definitely absent.
type ExistenceFilter interface {
	MightExist(ctx context.Context, domainID int64, code string) (bool, error)
	Add(ctx context.Context, domainID int64, code string) error
}

// L2Cache is the Redis tier (positive + negative caching).
type L2Cache interface {
	Get(ctx context.Context, domainID int64, code string) (model.Link, bool, error)
	Set(ctx context.Context, l model.Link, ttl time.Duration) error
	IsNegative(ctx context.Context, domainID int64, code string) (bool, error)
	SetNegative(ctx context.Context, domainID int64, code string, ttl time.Duration) error
}

// LinkRepo is the authoritative store (read replica).
type LinkRepo interface {
	Lookup(ctx context.Context, domainID int64, code string) (model.Link, bool, error)
}

// Resolver implements the multi-tier lookup: L1 → Bloom → Redis(+negative) → Postgres, with
// singleflight stampede protection and jittered backfill TTLs. Any tier may be nil (disabled),
// and errors from the cache tiers fail open to the next tier — the hot path keeps serving.
type Resolver struct {
	l1    L1Cache
	bloom ExistenceFilter
	l2    L2Cache
	repo  LinkRepo

	recTTL    time.Duration
	recJitter time.Duration
	negTTL    time.Duration

	sf      singleflight.Group
	metrics *observability.Metrics
}

// Options configures a Resolver.
type Options struct {
	L1            L1Cache
	Bloom         ExistenceFilter
	L2            L2Cache
	Repo          LinkRepo
	RecordTTL     time.Duration
	RecordJitter  time.Duration
	NegativeTTL   time.Duration
	Metrics       *observability.Metrics
}

// New builds a Resolver.
func New(o Options) *Resolver {
	return &Resolver{
		l1: o.L1, bloom: o.Bloom, l2: o.L2, repo: o.Repo,
		recTTL: o.RecordTTL, recJitter: o.RecordJitter, negTTL: o.NegativeTTL,
		metrics: o.Metrics,
	}
}

func composeKey(domainID int64, code string) string {
	return strconv.FormatInt(domainID, 10) + ":" + code
}

func (r *Resolver) cache(tier, result string) {
	if r.metrics != nil {
		r.metrics.CacheEvents.WithLabelValues(tier, result).Inc()
	}
}

func (r *Resolver) jittered(base, jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return base
	}
	return base + time.Duration(rand.Int64N(int64(jitter)))
}

// Resolve returns the link for (domainID, code), or ErrNotFound.
func (r *Resolver) Resolve(ctx context.Context, domainID int64, code string) (model.Link, Source, error) {
	key := composeKey(domainID, code)

	// Tier 1: in-process LRU.
	if r.l1 != nil {
		if l, ok := r.l1.Get(key); ok {
			r.cache("l1", "hit")
			return l, SourceL1, nil
		}
		r.cache("l1", "miss")
	}

	// Bloom: reject codes that definitely do not exist (absorbs enumeration before Redis/DB).
	if r.bloom != nil {
		if might, err := r.bloom.MightExist(ctx, domainID, code); err == nil && !might {
			r.cache("bloom", "reject")
			return model.Link{}, "", ErrNotFound
		}
	}

	// Tier 2: Redis positive + negative cache.
	if r.l2 != nil {
		l, ok, err := r.l2.Get(ctx, domainID, code)
		if err != nil {
			r.resolveErr()
		} else if ok {
			r.cache("redis", "hit")
			if r.l1 != nil {
				r.l1.Add(key, l)
			}
			return l, SourceRedis, nil
		} else {
			r.cache("redis", "miss")
			if neg, nerr := r.l2.IsNegative(ctx, domainID, code); nerr == nil && neg {
				return model.Link{}, "", ErrNotFound
			}
		}
	}

	// Tier 3: authoritative read, coalesced by singleflight so a hot miss hits the DB once.
	v, err, _ := r.sf.Do(key, func() (any, error) {
		if r.repo == nil {
			return nil, ErrNotFound
		}
		l, ok, lerr := r.repo.Lookup(ctx, domainID, code)
		if lerr != nil {
			return nil, lerr
		}
		if !ok {
			if r.l2 != nil {
				_ = r.l2.SetNegative(ctx, domainID, code, r.jittered(r.negTTL, r.negTTL/10))
			}
			return nil, ErrNotFound
		}
		if r.l2 != nil {
			_ = r.l2.Set(ctx, l, r.jittered(r.recTTL, r.recJitter))
		}
		if r.bloom != nil {
			_ = r.bloom.Add(ctx, domainID, code)
		}
		return l, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return model.Link{}, "", ErrNotFound
		}
		r.resolveErr()
		return model.Link{}, "", err
	}

	l := v.(model.Link)
	r.cache("pg", "hit")
	if r.l1 != nil {
		r.l1.Add(key, l)
	}
	return l, SourcePostgres, nil
}

func (r *Resolver) resolveErr() {
	if r.metrics != nil {
		r.metrics.ResolveErrors.Inc()
	}
}
