// services/redirect/internal/handler/limiter.go
package handler

import (
	"context"
	"time"

	"github.com/linkspire/linkspire/services/redirect/internal/cache"
)

// RedisLimiter implements AttemptLimiter with a per-link+client counter and a temporary
// lockout, both stored in Redis. Keys are provided by the caller (already anonymized).
type RedisLimiter struct {
	r           *cache.Redis
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
}

// NewRedisLimiter builds a limiter.
func NewRedisLimiter(r *cache.Redis, maxAttempts int, window, lockout time.Duration) *RedisLimiter {
	return &RedisLimiter{r: r, maxAttempts: maxAttempts, window: window, lockout: lockout}
}

// Allow returns false (with a Retry-After) while a lockout is active.
func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	locked, err := l.r.ExistsKey(ctx, key+":lock")
	if err != nil {
		return true, 0, nil // fail-open: don't lock users out on a Redis blip
	}
	if locked {
		ttl, _ := l.r.TTLOf(ctx, key+":lock")
		if ttl < 0 {
			ttl = l.lockout
		}
		return false, ttl, nil
	}
	return true, 0, nil
}

// RecordFailure increments the attempt counter and engages a lockout at the threshold.
func (l *RedisLimiter) RecordFailure(ctx context.Context, key string) (bool, error) {
	n, err := l.r.Incr(ctx, key+":att", l.window)
	if err != nil {
		return false, err
	}
	if int(n) >= l.maxAttempts {
		_ = l.r.SetString(ctx, key+":lock", "1", l.lockout)
		return true, nil
	}
	return false, nil
}

// Reset clears counters after a successful unlock.
func (l *RedisLimiter) Reset(ctx context.Context, key string) {
	_ = l.r.DelKeys(ctx, key+":att", key+":lock")
}
