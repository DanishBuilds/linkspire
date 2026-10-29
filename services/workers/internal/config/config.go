// services/workers/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Role            string // all | scanner | scraper | verifier | qr | retention | dsar | webhooks
	MetricsAddr     string
	PGDSN           string
	PGMaxConns      int32
	AssetDir        string
	PublicAssetBase string
	PollInterval    time.Duration
	URLHausURL      string
	GSBFile         string
}

func Load() (*Config, error) {
	c := &Config{
		Role:            getStr("WORKER_ROLE", "all"),
		MetricsAddr:     getStr("METRICS_ADDR", ":9092"),
		PGDSN:           getStr("PG_DSN", "postgresql://root@127.0.0.1:26257/linkspire?sslmode=disable"),
		PGMaxConns:      int32(getInt("PG_MAX_CONNS", 8)),
		AssetDir:        getStr("ASSET_DIR", os.TempDir()+"/linkspire-qr"),
		PublicAssetBase: strings.TrimRight(getStr("PUBLIC_ASSET_BASE", "/assets/qr"), "/"),
		PollInterval:    getDur("POLL_INTERVAL", 5*time.Second),
		URLHausURL:      getStr("URLHAUS_CSV", ""),
		GSBFile:         getStr("GSB_PREFIX_FILE", ""),
	}
	if c.PGDSN == "" {
		return nil, fmt.Errorf("PG_DSN is required")
	}
	return c, nil
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
func getDur(k string, d time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok {
		if x, err := time.ParseDuration(v); err == nil {
			return x
		}
	}
	return d
}
