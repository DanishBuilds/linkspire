// services/redirect/internal/cache/kv.go
package cache

import (
	"context"
	"time"
)

// Incr atomically increments a counter, setting the TTL on first creation. Used for
// per-link/per-client password attempt counters and rate-limit windows.
func (r *Redis) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := r.c.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		_ = r.c.Expire(ctx, key, ttl).Err()
	}
	return n, nil
}

// SetString stores a string value with a TTL (used for lockouts).
func (r *Redis) SetString(ctx context.Context, key, val string, ttl time.Duration) error {
	return r.c.Set(ctx, key, val, ttl).Err()
}

// ExistsKey reports whether key exists.
func (r *Redis) ExistsKey(ctx context.Context, key string) (bool, error) {
	n, err := r.c.Exists(ctx, key).Result()
	return n > 0, err
}

// TTLOf returns the remaining TTL of a key (used to compute Retry-After).
func (r *Redis) TTLOf(ctx context.Context, key string) (time.Duration, error) {
	return r.c.TTL(ctx, key).Result()
}

// DelKeys removes the given keys (used to reset counters/locks).
func (r *Redis) DelKeys(ctx context.Context, keys ...string) error {
	return r.c.Del(ctx, keys...).Err()
}
