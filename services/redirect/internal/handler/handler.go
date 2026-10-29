// services/redirect/internal/handler/handler.go
package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/linkspire/linkspire/services/redirect/internal/events"
	"github.com/linkspire/linkspire/services/redirect/internal/model"
	"github.com/linkspire/linkspire/services/redirect/internal/observability"
	"github.com/linkspire/linkspire/services/redirect/internal/privacy"
	"github.com/linkspire/linkspire/services/redirect/internal/resolver"
)

// LinkResolver is the resolve-chain dependency (satisfied by *resolver.Resolver).
type LinkResolver interface {
	Resolve(ctx context.Context, domainID int64, code string) (model.Link, resolver.Source, error)
}

// PasswordRepo fetches the Argon2id PHC string for a protected link.
type PasswordRepo interface {
	PasswordHash(ctx context.Context, domainID int64, code string) (string, bool, error)
}

// AttemptLimiter guards the password endpoint against brute force.
type AttemptLimiter interface {
	// Allow reports whether an attempt may proceed, and if locked, the retry-after duration.
	Allow(ctx context.Context, key string) (bool, time.Duration, error)
	// RecordFailure increments the failure counter and returns whether a lockout was triggered.
	RecordFailure(ctx context.Context, key string) (bool, error)
	// Reset clears counters on success.
	Reset(ctx context.Context, key string)
}

// Deps bundles the handler dependencies (dependency injection).
type Deps struct {
	Resolver LinkResolver
	Hosts    *HostResolver
	Emitter  events.Emitter
	Geo      privacy.GeoResolver
	Salt     privacy.SaltProvider
	PwRepo   PasswordRepo
	Limiter  AttemptLimiter
	Metrics  *observability.Metrics

	DefaultRedirectType int16
	PermanentCacheMaxAge time.Duration
	ResolveTimeout      time.Duration

	CookieSigningKey []byte
	CookieTTL        time.Duration
	SecureCookies    bool

	Argon2Memory  uint32
	Argon2Time    uint32
	Argon2Threads uint8
	PWMaxAttempts int
	PWLockout     time.Duration
}

// Handler serves the redirect hot path and the password-unlock flow.
type Handler struct {
	d Deps
}

// New builds a Handler.
func New(d Deps) *Handler {
	if d.Geo == nil {
		d.Geo = privacy.StubGeo{}
	}
	if d.Emitter == nil {
		d.Emitter = events.Nop{}
	}
	if d.Limiter == nil {
		d.Limiter = nopLimiter{}
	}
	if d.Salt == nil {
		d.Salt = privacy.StaticSalt("")
	}
	return &Handler{d: d}
}

// Routes returns the HTTP handler with all routes mounted.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "", http.StatusNoContent)
	})
	mux.HandleFunc("/", h.dispatch)
	return securityHeaders(mux)
}

// dispatch routes GET → redirect and POST → password submit for /{code}.
func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleRedirect(w, r)
	case http.MethodPost:
		h.handlePasswordSubmit(w, r)
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) observe(outcome *string, start time.Time) {
	if h.d.Metrics == nil {
		return
	}
	h.d.Metrics.RequestsTotal.WithLabelValues(*outcome).Inc()
	h.d.Metrics.RequestDuration.WithLabelValues(*outcome).Observe(time.Since(start).Seconds())
}

func extractCode(path string) string {
	code := strings.TrimPrefix(path, "/")
	if i := strings.IndexByte(code, '/'); i >= 0 {
		code = code[:i]
	}
	return code
}

func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	outcome := "not_found"
	defer h.observe(&outcome, start)

	code := extractCode(r.URL.Path)
	if code == "" {
		outcome = "root"
		h.serveNotFound(w) // per-domain root redirect is a management-plane setting (Phase 4/5)
		return
	}

	domainID, ok := h.d.Hosts.Resolve(r.Context(), r.Host)
	if !ok {
		outcome = "unknown_host"
		h.serveNotFound(w)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.d.ResolveTimeout)
	defer cancel()

	link, _, err := h.d.Resolver.Resolve(ctx, domainID, code)
	if err != nil {
		if err == resolver.ErrNotFound {
			outcome = "not_found"
			h.serveNotFound(w)
			return
		}
		// Dependency failure (e.g. DB unreachable). Degrade with 503, never a wrong redirect.
		outcome = "error"
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	switch link.EffectiveStatus(time.Now()) {
	case model.StatusActive:
		outcome = h.doRedirect(w, r, link)
	case model.StatusPasswordProtected:
		if h.hasValidUnlockCookie(r, link.DomainID, link.Code) {
			outcome = h.doRedirect(w, r, link)
			return
		}
		outcome = "password_prompt"
		h.servePasswordPrompt(w, link.Code, "")
	case model.StatusFlaggedMalicious:
		outcome = "warning"
		h.serveWarning(w, link.Destination)
	case model.StatusExpired:
		outcome = "expired"
		h.serveGone(w)
	case model.StatusDisabled:
		outcome = "disabled"
		h.serveGone(w)
	default:
		outcome = "not_found"
		h.serveNotFound(w)
	}
}

// doRedirect writes the redirect response and emits the async click event. Returns the metric
// outcome label. It NEVER blocks on analytics.
func (h *Handler) doRedirect(w http.ResponseWriter, r *http.Request, link model.Link) string {
	dest := link.Destination
	// Defense-in-depth against response-splitting (DB constraint already enforces http(s)).
	if strings.ContainsAny(dest, "\r\n") {
		http.Error(w, "bad destination", http.StatusBadGateway)
		return "error"
	}

	status := int(link.RedirectType)
	switch status {
	case 301, 302, 307, 308:
	default:
		status = int(h.d.DefaultRedirectType)
	}

	// Caching policy (ADR-004): permanent codes may be cached; temporary must reach origin
	// each hit so analytics are counted and state is re-evaluated.
	if status == 301 || status == 308 {
		w.Header().Set("Cache-Control", "public, max-age="+itoa(int(h.d.PermanentCacheMaxAge.Seconds())))
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Location", dest)
	w.WriteHeader(status)

	h.emitClick(r, link, uint16(status))
	return "redirect_" + itoa(status)
}

// emitClick derives privacy-safe signals in memory, enqueues the event, and discards the IP.
func (h *Handler) emitClick(r *http.Request, link model.Link, statusServed uint16) {
	ip := privacy.ClientIP(r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP"), r.RemoteAddr)
	country, region := h.d.Geo.Lookup(ip)
	device, browser, os, isBot := privacy.ClassifyUA(r.UserAgent())
	visitor := privacy.VisitorHash(ip, h.d.Salt.Current())
	// ip goes out of scope here and is never stored or logged.

	source := "web"
	if s := r.URL.Query().Get("s"); s == "qr" {
		source = "qr"
	}
	var bot uint8
	if isBot {
		bot = 1
	}
	h.d.Emitter.Emit(events.ClickEvent{
		EventTime:      time.Now().Unix(),
		DomainID:       link.DomainID,
		Code:           link.Code,
		WorkspaceID:    link.WorkspaceID,
		Country:        country,
		Region:         region,
		DeviceClass:    device,
		BrowserFamily:  browser,
		OSFamily:       os,
		ReferrerDomain: privacy.ReferrerDomain(r.Referer()),
		Source:         source,
		IsBot:          bot,
		StatusServed:   statusServed,
		VisitorHash:    visitor,
	})
}

// ---- response helpers ----

func (h *Handler) serveNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = templates.ExecuteTemplate(w, "notfound.html", nil)
}

func (h *Handler) serveGone(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	_ = templates.ExecuteTemplate(w, "notfound.html", nil)
}

func (h *Handler) serveWarning(w http.ResponseWriter, dest string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.WriteHeader(http.StatusOK)
	_ = templates.ExecuteTemplate(w, "warning.html", warningData{Destination: dest})
}

// securityHeaders applies baseline headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// itoa avoids importing strconv at call sites for small ints.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
