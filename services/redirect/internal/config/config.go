// services/redirect/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the typed configuration for the redirect service. Every field has a sane default;
// Validate() is called at startup and the process refuses to boot on an invalid combination.
type Config struct {
	Env             string
	HTTPAddr        string
	MetricsAddr     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	ResolveTimeout  time.Duration // per-request budget for the cache/DB resolve chain

	// L1 in-process cache.
	L1Size int
	L1TTL  time.Duration

	// Redis (L2). Multiple addrs → cluster client; single addr → standalone.
	RedisEnabled  bool
	RedisAddrs    []string
	RedisPassword string
	RedisTTL      time.Duration
	RedisTTLJitter time.Duration
	NegativeTTL   time.Duration

	// Bloom existence filter.
	BloomEnabled  bool
	BloomUseRedis bool // true → RedisBloom (shared); false → in-process bloom
	BloomCapacity uint
	BloomFP       float64
	BloomRedisKey string

	// Postgres / CockroachDB read replica.
	PGEnabled bool
	PGDSN     string
	PGMaxConns int32

	// Kafka / Redpanda click-event emission.
	KafkaEnabled    bool
	KafkaBrokers    []string
	KafkaTopic      string
	EventBufferSize int

	// Redirect behavior.
	DefaultRedirectType int16
	DefaultDomainID     int64 // fallback domain when host is unmapped (dev/single-domain)
	PermanentCacheMaxAge time.Duration

	// Password unlock.
	CookieSigningKey []byte
	CookieTTL        time.Duration
	Argon2Memory     uint32
	Argon2Time       uint32
	Argon2Threads    uint8
	PWMaxAttempts    int
	PWLockout        time.Duration

	// Privacy.
	StaticVisitorSalt string // dev only; prod uses a Redis-backed rotating salt
	GeoDBPath         string

	// Tracing.
	TracingEnabled bool
	OTLPEndpoint   string
	ServiceName    string
}

// Load reads configuration from the environment and applies defaults.
func Load() (*Config, error) {
	c := &Config{
		Env:                  getStr("ENV", "dev"),
		HTTPAddr:             getStr("HTTP_ADDR", ":8080"),
		MetricsAddr:          getStr("METRICS_ADDR", ":9090"),
		ReadTimeout:          getDur("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:         getDur("WRITE_TIMEOUT", 5*time.Second),
		ShutdownTimeout:      getDur("SHUTDOWN_TIMEOUT", 20*time.Second),
		ResolveTimeout:       getDur("RESOLVE_TIMEOUT", 250*time.Millisecond),
		L1Size:               getInt("L1_SIZE", 100_000),
		L1TTL:                getDur("L1_TTL", 30*time.Second),
		RedisEnabled:         getBool("REDIS_ENABLED", false),
		RedisAddrs:           getCSV("REDIS_ADDRS", []string{"127.0.0.1:6379"}),
		RedisPassword:        getStr("REDIS_PASSWORD", ""),
		RedisTTL:             getDur("REDIS_TTL", 3600*time.Second),
		RedisTTLJitter:       getDur("REDIS_TTL_JITTER", 360*time.Second),
		NegativeTTL:          getDur("NEGATIVE_TTL", 60*time.Second),
		// Bloom defaults OFF: it is only correct when the filter contains ALL existing codes.
		// In production enable BLOOM_USE_REDIS so the shared RedisBloom (maintained by the
		// creation pipeline) is used; an empty in-process filter would reject valid codes.
		BloomEnabled:         getBool("BLOOM_ENABLED", false),
		BloomUseRedis:        getBool("BLOOM_USE_REDIS", false),
		BloomCapacity:        uint(getInt("BLOOM_CAPACITY", 2_000_000_000)),
		BloomFP:              getFloat("BLOOM_FP", 0.001),
		BloomRedisKey:        getStr("BLOOM_REDIS_KEY", "bf:codes"),
		PGEnabled:            getBool("PG_ENABLED", false),
		PGDSN:                getStr("PG_DSN", "postgresql://root@127.0.0.1:26257/linkspire?sslmode=disable"),
		PGMaxConns:           int32(getInt("PG_MAX_CONNS", 16)),
		KafkaEnabled:         getBool("KAFKA_ENABLED", false),
		KafkaBrokers:         getCSV("KAFKA_BROKERS", []string{"127.0.0.1:9092"}),
		KafkaTopic:           getStr("KAFKA_TOPIC", "click_events"),
		EventBufferSize:      getInt("EVENT_BUFFER_SIZE", 65536),
		DefaultRedirectType:  int16(getInt("DEFAULT_REDIRECT_TYPE", 302)),
		DefaultDomainID:      int64(getInt("DEFAULT_DOMAIN_ID", 0)),
		PermanentCacheMaxAge: getDur("PERMANENT_CACHE_MAX_AGE", 24*time.Hour),
		CookieSigningKey:     []byte(getStr("COOKIE_SIGNING_KEY", "dev-insecure-cookie-key-change-me")),
		CookieTTL:            getDur("COOKIE_TTL", 30*time.Minute),
		Argon2Memory:         uint32(getInt("ARGON2_MEMORY_KIB", 65536)),
		Argon2Time:           uint32(getInt("ARGON2_TIME", 3)),
		Argon2Threads:        uint8(getInt("ARGON2_THREADS", 2)),
		PWMaxAttempts:        getInt("PW_MAX_ATTEMPTS", 5),
		PWLockout:            getDur("PW_LOCKOUT", 15*time.Minute),
		StaticVisitorSalt:    getStr("STATIC_VISITOR_SALT", "dev-visitor-salt"),
		GeoDBPath:            getStr("GEO_DB_PATH", ""),
		TracingEnabled:       getBool("TRACING_ENABLED", false),
		OTLPEndpoint:         getStr("OTLP_ENDPOINT", "localhost:4318"),
		ServiceName:          getStr("SERVICE_NAME", "linkspire-redirect"),
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate enforces cross-field invariants.
func (c *Config) Validate() error {
	if c.HTTPAddr == "" {
		return fmt.Errorf("HTTP_ADDR must not be empty")
	}
	switch c.DefaultRedirectType {
	case 301, 302, 307, 308:
	default:
		return fmt.Errorf("DEFAULT_REDIRECT_TYPE must be one of 301/302/307/308, got %d", c.DefaultRedirectType)
	}
	if c.RedisEnabled && len(c.RedisAddrs) == 0 {
		return fmt.Errorf("REDIS_ENABLED but REDIS_ADDRS is empty")
	}
	if c.PGEnabled && c.PGDSN == "" {
		return fmt.Errorf("PG_ENABLED but PG_DSN is empty")
	}
	if c.KafkaEnabled && len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_ENABLED but KAFKA_BROKERS is empty")
	}
	if c.BloomUseRedis && !c.RedisEnabled {
		return fmt.Errorf("BLOOM_USE_REDIS requires REDIS_ENABLED")
	}
	if c.L1Size <= 0 {
		return fmt.Errorf("L1_SIZE must be > 0")
	}
	if c.ResolveTimeout <= 0 {
		return fmt.Errorf("RESOLVE_TIMEOUT must be > 0")
	}
	if len(c.CookieSigningKey) < 16 {
		return fmt.Errorf("COOKIE_SIGNING_KEY must be at least 16 bytes")
	}
	if c.Env == "prod" && string(c.CookieSigningKey) == "dev-insecure-cookie-key-change-me" {
		return fmt.Errorf("refusing to start in prod with the default COOKIE_SIGNING_KEY")
	}
	return nil
}

// ---- env helpers ----

func getStr(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}

func getInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(k string, def float64) float64 {
	if v, ok := os.LookupEnv(k); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getBool(k string, def bool) bool {
	if v, ok := os.LookupEnv(k); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getDur(k string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getCSV(k string, def []string) []string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return def
}
