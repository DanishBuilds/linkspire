// services/api/internal/httpserver/analytics_ch.go
// Privacy-safe analytics read from ClickHouse via its HTTP interface. No raw IP
// or user agent exists in the store; only derived daily aggregates. Failures
// degrade to an empty result so they never breach the API availability SLO.
package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var chCode = func(s string) string {
	// Codes are Base62 or validated aliases; keep only safe characters before
	// interpolation (defense in depth on top of the route already resolving a link).
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Server) clickhouseRollups(ctx context.Context, domainID int64, code string, days int) ([]map[string]any, error) {
	if days <= 0 || days > 760 {
		days = 90
	}
	sql := fmt.Sprintf(`SELECT toString(event_date) AS date, country_code, device_class,
		browser_family, os_family, referrer_domain, source,
		toInt64(clicks) AS clicks, toInt64(unique_visitors) AS unique_visitors
		FROM linkspire.clicks_daily_final
		WHERE domain_id = %d AND code = '%s' AND event_date >= today() - %d
		ORDER BY event_date FORMAT JSON`, domainID, chCode(code), days)

	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, s.cfg.ClickHouseURL, strings.NewReader(sql))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := (&http.Client{Timeout: 4 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("clickhouse %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}
