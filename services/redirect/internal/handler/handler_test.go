// services/redirect/internal/handler/handler_test.go
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
	"github.com/linkspire/linkspire/services/redirect/internal/resolver"
)

type fakeResolver struct {
	link model.Link
	err  error
}

func (f fakeResolver) Resolve(context.Context, int64, string) (model.Link, resolver.Source, error) {
	return f.link, resolver.SourceL1, f.err
}

func newTestHandler(link model.Link, err error) *Handler {
	return New(Deps{
		Resolver:            fakeResolver{link: link, err: err},
		Hosts:               NewHostResolver(nil, 1, time.Minute), // default domain 1 for any host
		DefaultRedirectType: 302,
		ResolveTimeout:      time.Second,
		CookieSigningKey:    []byte("0123456789abcdef-signing-key"),
		CookieTTL:           30 * time.Minute,
	})
}

func do(h *Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Host = "lnk.example"
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)
	return rr
}

func TestRedirectActive302(t *testing.T) {
	h := newTestHandler(model.Link{DomainID: 1, Code: "abc", Destination: "https://example.com/x", Status: model.StatusActive, RedirectType: 302}, nil)
	rr := do(h, http.MethodGet, "/abc")
	if rr.Code != http.StatusFound {
		t.Fatalf("code=%d want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "https://example.com/x" {
		t.Fatalf("Location=%q", loc)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control=%q want no-store", cc)
	}
}

func TestRedirectPermanent301Cacheable(t *testing.T) {
	h := newTestHandler(model.Link{DomainID: 1, Code: "abc", Destination: "https://example.com/x", Status: model.StatusActive, RedirectType: 301}, nil)
	rr := do(h, http.MethodGet, "/abc")
	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("code=%d want 301", rr.Code)
	}
	if cc := rr.Header().Get("Cache-Control"); !strings.HasPrefix(cc, "public") {
		t.Fatalf("Cache-Control=%q want public for permanent", cc)
	}
}

func TestNotFound(t *testing.T) {
	h := newTestHandler(model.Link{}, resolver.ErrNotFound)
	rr := do(h, http.MethodGet, "/missing")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code=%d want 404", rr.Code)
	}
}

func TestFlaggedServesWarning(t *testing.T) {
	h := newTestHandler(model.Link{DomainID: 1, Code: "bad", Destination: "https://evil.example", Status: model.StatusFlaggedMalicious, RedirectType: 302}, nil)
	rr := do(h, http.MethodGet, "/bad")
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d want 200 interstitial", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "may be unsafe") {
		t.Fatal("expected warning interstitial body")
	}
	// Destination must NOT be auto-followed (no Location header).
	if rr.Header().Get("Location") != "" {
		t.Fatal("warning page must not set Location")
	}
}

func TestExpiredGone(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	h := newTestHandler(model.Link{DomainID: 1, Code: "old", Destination: "https://example.com", Status: model.StatusActive, RedirectType: 302, ExpiresAt: &past}, nil)
	rr := do(h, http.MethodGet, "/old")
	if rr.Code != http.StatusGone {
		t.Fatalf("code=%d want 410", rr.Code)
	}
}

func TestPasswordPromptSetsCSRF(t *testing.T) {
	h := newTestHandler(model.Link{DomainID: 1, Code: "sec", Destination: "https://example.com", Status: model.StatusPasswordProtected, RedirectType: 302}, nil)
	rr := do(h, http.MethodGet, "/sec")
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d want 200 prompt", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "password protected") {
		t.Fatal("expected password interstitial")
	}
	var hasCSRF bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == csrfCookie {
			hasCSRF = true
		}
	}
	if !hasCSRF {
		t.Fatal("expected CSRF cookie to be set")
	}
}

func TestResolverErrorDegrades503(t *testing.T) {
	h := newTestHandler(model.Link{}, context.DeadlineExceeded)
	rr := do(h, http.MethodGet, "/x")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d want 503 on dependency error", rr.Code)
	}
}
