// services/redirect/internal/handler/password.go
package handler

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
	"github.com/linkspire/linkspire/services/redirect/internal/privacy"
	"github.com/linkspire/linkspire/services/redirect/internal/resolver"
)

const (
	unlockCookie = "ls_pw"
	csrfCookie   = "ls_csrf"
)

// nopLimiter allows everything (used when Redis is disabled). Password verification still runs.
type nopLimiter struct{}

func (nopLimiter) Allow(context.Context, string) (bool, time.Duration, error) { return true, 0, nil }
func (nopLimiter) RecordFailure(context.Context, string) (bool, error)        { return false, nil }
func (nopLimiter) Reset(context.Context, string)                              {}

func (h *Handler) servePasswordPrompt(w http.ResponseWriter, code, errMsg string) {
	token := randToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/" + code,
		HttpOnly: true,
		Secure:   h.d.SecureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   600,
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'")
	if errMsg != "" {
		w.WriteHeader(http.StatusUnauthorized)
	}
	_ = templates.ExecuteTemplate(w, "password.html", passwordData{Code: code, CSRFToken: token, Error: errMsg})
}

func (h *Handler) handlePasswordSubmit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	outcome := "password_fail"
	defer h.observe(&outcome, start)

	code := extractCode(r.URL.Path)
	domainID, ok := h.d.Hosts.Resolve(r.Context(), r.Host)
	if code == "" || !ok {
		outcome = "not_found"
		h.serveNotFound(w)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.d.ResolveTimeout)
	defer cancel()

	link, _, err := h.d.Resolver.Resolve(ctx, domainID, code)
	if err != nil {
		if errors.Is(err, resolver.ErrNotFound) {
			outcome = "not_found"
			h.serveNotFound(w)
			return
		}
		outcome = "error"
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if link.EffectiveStatus(time.Now()) != model.StatusPasswordProtected {
		// Not (or no longer) password protected — fall through to a normal GET.
		outcome = "not_found"
		h.serveNotFound(w)
		return
	}

	if err := r.ParseForm(); err != nil {
		outcome = "bad_request"
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// CSRF: double-submit cookie.
	cookie, cerr := r.Cookie(csrfCookie)
	formToken := r.PostFormValue("csrf_token")
	if cerr != nil || formToken == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) != 1 {
		outcome = "csrf_fail"
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}

	ip := privacy.ClientIP(r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP"), r.RemoteAddr)
	fp := privacy.Fingerprint(ip, "")
	limiterKey := fmt.Sprintf("pw:%d:%s:%s", domainID, code, fp)

	allowed, retryAfter, _ := h.d.Limiter.Allow(ctx, limiterKey)
	if !allowed {
		outcome = "password_locked"
		h.pwMetric("locked")
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		http.Error(w, "too many attempts, try again later", http.StatusTooManyRequests)
		return
	}

	phc, found, perr := h.pwHash(ctx, domainID, code)
	if perr != nil {
		outcome = "error"
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if !found {
		outcome = "not_found"
		h.serveNotFound(w)
		return
	}

	if verifyArgon2id(r.PostFormValue("password"), phc) {
		h.d.Limiter.Reset(ctx, limiterKey)
		h.setUnlockCookie(w, domainID, code)
		h.pwMetric("ok")
		outcome = "password_ok"
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Location", link.Destination)
		w.WriteHeader(http.StatusFound)
		h.emitClick(r, link, uint16(http.StatusFound))
		return
	}

	// Failure: progressive delay + lockout.
	locked, _ := h.d.Limiter.RecordFailure(ctx, limiterKey)
	time.Sleep(300 * time.Millisecond)
	if locked {
		outcome = "password_locked"
		h.pwMetric("locked")
	} else {
		h.pwMetric("fail")
	}
	h.servePasswordPrompt(w, code, "Incorrect password.")
}

func (h *Handler) pwHash(ctx context.Context, domainID int64, code string) (string, bool, error) {
	if h.d.PwRepo == nil {
		return "", false, errors.New("password store unavailable")
	}
	return h.d.PwRepo.PasswordHash(ctx, domainID, code)
}

func (h *Handler) pwMetric(result string) {
	if h.d.Metrics != nil {
		h.d.Metrics.PasswordAttempts.WithLabelValues(result).Inc()
	}
}

// ---- signed unlock cookie ----

func (h *Handler) setUnlockCookie(w http.ResponseWriter, domainID int64, code string) {
	exp := time.Now().Add(h.d.CookieTTL)
	payload := fmt.Sprintf("%d:%s:%d", domainID, code, exp.Unix())
	value := b64(payload) + "." + b64(string(h.sign(payload)))
	http.SetCookie(w, &http.Cookie{
		Name:     unlockCookie,
		Value:    value,
		Path:     "/" + code,
		HttpOnly: true,
		Secure:   h.d.SecureCookies,
		SameSite: http.SameSiteStrictMode,
		Expires:  exp,
		MaxAge:   int(h.d.CookieTTL.Seconds()),
	})
}

func (h *Handler) hasValidUnlockCookie(r *http.Request, domainID int64, code string) bool {
	c, err := r.Cookie(unlockCookie)
	if err != nil {
		return false
	}
	dot := strings.IndexByte(c.Value, '.')
	if dot < 0 {
		return false
	}
	payload, err1 := unb64(c.Value[:dot])
	mac, err2 := unb64(c.Value[dot+1:])
	if err1 != nil || err2 != nil {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(mac), h.sign(payload)) != 1 {
		return false
	}
	// payload = "domainID:code:exp"
	parts := strings.Split(payload, ":")
	if len(parts) != 3 {
		return false
	}
	gotDomain, _ := strconv.ParseInt(parts[0], 10, 64)
	exp, _ := strconv.ParseInt(parts[2], 10, 64)
	if gotDomain != domainID || parts[1] != code {
		return false
	}
	return time.Now().Unix() < exp
}

func (h *Handler) sign(payload string) []byte {
	mac := hmac.New(sha256.New, h.d.CookieSigningKey)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// ---- Argon2id verification ----

// verifyArgon2id parses a PHC-encoded Argon2id hash and verifies the password in constant time.
// It never logs the password.
func verifyArgon2id(password, phc string) bool {
	parts := strings.Split(phc, "$")
	// ["", "argon2id", "v=19", "m=65536,t=3,p=2", saltB64, hashB64]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var m uint32
	var t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ---- small helpers ----

func randToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func b64(s string) string   { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
func unb64(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	return string(b), err
}
