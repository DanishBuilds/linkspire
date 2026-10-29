// services/redirect/internal/cache/redis.go
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
)

// Redis is the L2 shared cache: positive records, negative cache, and (optionally) the
// RedisBloom existence filter and the rotating visitor salt. Uses a UniversalClient so one
// addr → standalone and multiple addrs → cluster.
type Redis struct {
	c redis.UniversalClient
}

// NewRedis constructs the client. ping verifies connectivity at startup.
func NewRedis(ctx context.Context, addrs []string, password string) (*Redis, error) {
	c := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        addrs,
		Password:     password,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		PoolSize:     64,
	})
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.Ping(pctx).Err(); err != nil {
		return nil, err
	}
	return &Redis{c: c}, nil
}

func recordKey(domainID int64, code string) string {
	b := appendInt([]byte("l:"), domainID)
	b = append(b, ':')
	b = append(b, code...)
	return string(b)
}

func negativeKey(domainID int64, code string) string {
	b := appendInt([]byte("n:"), domainID)
	b = append(b, ':')
	b = append(b, code...)
	return string(b)
}

// Get fetches a positive record. ok=false on cache miss (redis.Nil).
func (r *Redis) Get(ctx context.Context, domainID int64, code string) (model.Link, bool, error) {
	var l model.Link
	data, err := r.c.Get(ctx, recordKey(domainID, code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return l, false, nil
	}
	if err != nil {
		return l, false, err
	}
	if err := json.Unmarshal(data, &l); err != nil {
		return l, false, err
	}
	return l, true, nil
}

// Set stores a positive record with the given TTL (caller adds jitter).
func (r *Redis) Set(ctx context.Context, l model.Link, ttl time.Duration) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return r.c.Set(ctx, recordKey(l.DomainID, l.Code), data, ttl).Err()
}

// IsNegative reports whether a recent negative (not-found) result is cached.
func (r *Redis) IsNegative(ctx context.Context, domainID int64, code string) (bool, error) {
	n, err := r.c.Exists(ctx, negativeKey(domainID, code)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetNegative caches a not-found result briefly to absorb enumeration (caller adds jitter).
func (r *Redis) SetNegative(ctx context.Context, domainID int64, code string, ttl time.Duration) error {
	return r.c.Set(ctx, negativeKey(domainID, code), "1", ttl).Err()
}

// Delete removes the positive + negative cache entries (cache-bust on edit/delete/DSAR).
func (r *Redis) Delete(ctx context.Context, domainID int64, code string) error {
	return r.c.Del(ctx, recordKey(domainID, code), negativeKey(domainID, code)).Err()
}

// Ping is used by readiness probes.
func (r *Redis) Ping(ctx context.Context) error { return r.c.Ping(ctx).Err() }

// Close releases the client.
func (r *Redis) Close() error { return r.c.Close() }

// ---- RedisBloom existence filter (shared) ----

// RedisBloom implements the existence-filter interface against a shared RedisBloom key.
type RedisBloom struct {
	c   redis.UniversalClient
	key string
}

// NewRedisBloom wraps an existing client. Reserve the filter out-of-band once with
// `BF.RESERVE <key> <fp> <capacity>` (see capacity doc); this type only tests/adds.
func NewRedisBloom(r *Redis, key string) *RedisBloom {
	return &RedisBloom{c: r.c, key: key}
}

func (b *RedisBloom) MightExist(ctx context.Context, domainID int64, code string) (bool, error) {
	res, err := b.c.Do(ctx, "BF.EXISTS", b.key, string(bloomKey(domainID, code))).Int()
	if err != nil {
		return true, err // fail-open: never reject on bloom error
	}
	return res == 1, nil
}

func (b *RedisBloom) Add(ctx context.Context, domainID int64, code string) error {
	return b.c.Do(ctx, "BF.ADD", b.key, string(bloomKey(domainID, code))).Err()
}

// ---- rotating visitor salt ----

// Salt fetches the current daily visitor salt. The redirect service uses it to hash a
// visitor's IP in memory, then discards the IP. Returns "" if unset.
func (r *Redis) Salt(ctx context.Context) (string, error) {
	v, err := r.c.Get(ctx, "salt:visitor:current").Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return v, err
}
