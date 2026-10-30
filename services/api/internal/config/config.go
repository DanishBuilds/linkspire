// services/api/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is validated at process start.
type Config struct {
	Env             string
	HTTPAddr        string
	MetricsAddr     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	PGDSN      string
	PGMaxConns int32

	RedisEnabled  bool
	RedisAddrs    []string
	RedisPassword string

	KafkaEnabled bool
	KafkaBrokers []string
	KafkaTopic   string

	TokenKey          []byte
	TokenTTL          time.Duration
	FeistelKey        []byte
	CookieSecure      bool
	PublicBaseURL     string // e.g. https://lnk.linkspire.io
	DefaultDomainID   int64
	DefaultDomainHost string

	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCPostLoginURL string

	WebAuthnRPID     string
	WebAuthnRPOrigin string
	WebAuthnRPName   string

	QRPublicBase string
	QRAssetDir   string // local QR files written by workers (basename keys in DB)

	GSBFile       string
	ClickHouseURL string
	AskToken      string

	Argon2DisabledForTest bool // never set in prod; speeds tests
}

func Load() (*Config, error) {
	c := &Config{
		Env:               getStr("ENV", "dev"),
		HTTPAddr:          getStr("HTTP_ADDR", ":8081"),
		MetricsAddr:       getStr("METRICS_ADDR", ":9091"),
		ReadTimeout:       getDur("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      getDur("WRITE_TIMEOUT", 15*time.Second),
		ShutdownTimeout:   getDur("SHUTDOWN_TIMEOUT", 20*time.Second),
		PGDSN:             getStr("PG_DSN", "postgresql://root@127.0.0.1:26257/linkspire?sslmode=disable"),
		PGMaxConns:        int32(getInt("PG_MAX_CONNS", 32)),
		RedisEnabled:      getBool("REDIS_ENABLED", false),
		RedisAddrs:        csv("REDIS_ADDRS", []string{"127.0.0.1:6379"}),
		RedisPassword:     getStr("REDIS_PASSWORD", ""),
		KafkaEnabled:      getBool("KAFKA_ENABLED", false),
		KafkaBrokers:      csv("KAFKA_BROKERS", []string{"127.0.0.1:9092"}),
		KafkaTopic:        getStr("KAFKA_TOPIC", "link_jobs"),
		TokenKey:          []byte(getStr("TOKEN_SIGNING_KEY", "dev-insecure-token-key-change")),
		TokenTTL:          getDur("TOKEN_TTL", 12*time.Hour),
		FeistelKey:        []byte(getStr("FEISTEL_KEY", "dev-insecure-feistel-key-chg")),
		CookieSecure:      getStr("ENV", "dev") == "prod",
		PublicBaseURL:     strings.TrimRight(getStr("PUBLIC_BASE_URL", "https://lnk.linkspire.io"), "/"),
		DefaultDomainID:   int64(getInt("DEFAULT_DOMAIN_ID", 1)),
		DefaultDomainHost: getStr("DEFAULT_DOMAIN_HOST", "lnk.linkspire.io"),
		OIDCIssuer:        getStr("OIDC_ISSUER", ""),
		OIDCClientID:      getStr("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:  getStr("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:   getStr("OIDC_REDIRECT_URL", ""),
		OIDCPostLoginURL:  getStr("OIDC_POST_LOGIN_URL", ""),
		WebAuthnRPID:      getStr("WEBAUTHN_RP_ID", ""),
		WebAuthnRPOrigin:  getStr("WEBAUTHN_RP_ORIGIN", ""),
		WebAuthnRPName:    getStr("WEBAUTHN_RP_NAME", "Linkspire"),
		QRPublicBase:      getStr("QR_PUBLIC_BASE", ""),
		QRAssetDir:        getStr("QR_ASSET_DIR", ""),
		GSBFile:           getStr("GSB_PREFIX_FILE", ""),
		ClickHouseURL:     getStr("CLICKHOUSE_URL", ""),
		AskToken:          getStr("TLS_ASK_TOKEN", "dev-ask-token"),
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) Validate() error {
	if c.PGDSN == "" {
		return fmt.Errorf("PG_DSN is required")
	}
	if len(c.TokenKey) < 16 {
		return fmt.Errorf("TOKEN_SIGNING_KEY must be at least 16 bytes")
	}
	if len(c.FeistelKey) < 16 {
		return fmt.Errorf("FEISTEL_KEY must be at least 16 bytes")
	}
	if c.Env == "prod" && string(c.TokenKey) == "dev-insecure-token-key-change" {
		return fmt.Errorf("refusing to start in prod with the default TOKEN_SIGNING_KEY")
	}
	return nil
}

func getStr(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}
func getInt(k string, d int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}
func getBool(k string, d bool) bool {
	if v, ok := os.LookupEnv(k); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return d
}
func getDur(k string, d time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok {
		if x, err := time.ParseDuration(v); err == nil {
			return x
		}
	}
	return d
}
func csv(k string, d []string) []string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		p := strings.Split(v, ",")
		for i := range p {
			p[i] = strings.TrimSpace(p[i])
		}
		return p
	}
	return d
}
