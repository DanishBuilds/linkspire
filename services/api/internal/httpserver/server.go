// services/api/internal/httpserver/server.go
package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/linkspire/linkspire/pkg/argon2id"
	"github.com/linkspire/linkspire/pkg/idgen"
	"github.com/linkspire/linkspire/pkg/problem"
	"github.com/linkspire/linkspire/pkg/scan"
	"github.com/linkspire/linkspire/pkg/slug"
	"github.com/linkspire/linkspire/pkg/urlpolicy"
	"github.com/linkspire/linkspire/services/api/internal/auth"
	"github.com/linkspire/linkspire/services/api/internal/config"
	"github.com/linkspire/linkspire/services/api/internal/repo"
)

type ctxKey int

const principalKey ctxKey = 1

func PrincipalFrom(r *http.Request) auth.Principal {
	p, _ := r.Context().Value(principalKey).(auth.Principal)
	return p
}

// Server is the management API.
type Server struct {
	cfg   *config.Config
	repo  *repo.Repo
	alloc *idgen.Allocator
	gsb   *scan.HashPrefixDB
	log   *slog.Logger
}

func New(cfg *config.Config, r *repo.Repo, alloc *idgen.Allocator, gsb *scan.HashPrefixDB, log *slog.Logger) *Server {
	return &Server{cfg: cfg, repo: r, alloc: alloc, gsb: gsb, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/register", s.register)
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("POST /v1/auth/logout", s.logout)
	mux.HandleFunc("GET /v1/auth/me", s.authn(s.me))
	mux.HandleFunc("POST /v1/auth/mfa/totp/setup", s.authn(s.totpSetup))
	mux.HandleFunc("POST /v1/auth/mfa/totp/enable", s.authn(s.totpEnable))
	mux.HandleFunc("GET /v1/auth/oidc/start", s.oidcStart)
	mux.HandleFunc("GET /v1/auth/oidc/callback", s.oidcCallback)

	mux.HandleFunc("GET /v1/workspaces", s.authn(s.listWorkspaces))
	mux.HandleFunc("GET /v1/workspaces/{id}/members", s.authn(s.requireWS(s.listMembers)))
	mux.HandleFunc("PUT /v1/workspaces/{id}/members/{uid}", s.authn(s.requireWS(s.putMember)))

	mux.HandleFunc("POST /v1/links", s.authn(s.requireWS(s.createLink)))
	mux.HandleFunc("POST /v1/links/bulk", s.authn(s.requireWS(s.bulkLinks)))
	mux.HandleFunc("GET /v1/links", s.authn(s.requireWS(s.listLinks)))
	mux.HandleFunc("GET /v1/links/{domainId}/{code}", s.authn(s.requireWS(s.getLink)))
	mux.HandleFunc("PATCH /v1/links/{domainId}/{code}", s.authn(s.requireWS(s.patchLink)))
	mux.HandleFunc("DELETE /v1/links/{domainId}/{code}", s.authn(s.requireWS(s.deleteLink)))

	mux.HandleFunc("POST /v1/domains", s.authn(s.requireWS(s.createDomain)))
	mux.HandleFunc("GET /v1/domains", s.authn(s.requireWS(s.listDomains)))
	mux.HandleFunc("GET /v1/domains/{id}", s.authn(s.requireWS(s.getDomain)))
	mux.HandleFunc("PATCH /v1/domains/{id}", s.authn(s.requireWS(s.patchDomain)))

	mux.HandleFunc("POST /v1/links/{domainId}/{code}/qr", s.authn(s.requireWS(s.requestQR)))
	mux.HandleFunc("POST /v1/api-keys", s.authn(s.requireWS(s.createKey)))
	mux.HandleFunc("GET /v1/api-keys", s.authn(s.requireWS(s.listKeys)))
	mux.HandleFunc("DELETE /v1/api-keys/{id}", s.authn(s.requireWS(s.revokeKey)))

	mux.HandleFunc("POST /v1/webhooks", s.authn(s.requireWS(s.createWebhook)))
	mux.HandleFunc("GET /v1/webhooks", s.authn(s.requireWS(s.listWebhooks)))
	mux.HandleFunc("DELETE /v1/webhooks/{id}", s.authn(s.requireWS(s.deleteWebhook)))

	mux.HandleFunc("GET /v1/analytics/links/{domainId}/{code}", s.authn(s.requireWS(s.analytics)))

	mux.HandleFunc("POST /v1/privacy/requests", s.authn(s.createDSAR))
	mux.HandleFunc("GET /v1/privacy/requests", s.authn(s.listDSAR))
	mux.HandleFunc("GET /v1/privacy/requests/{id}", s.authn(s.getDSAR))
	mux.HandleFunc("POST /v1/privacy/do-not-sell", s.authn(s.doNotSell))

	mux.HandleFunc("GET /internal/tls/ask", s.tlsAsk)
	mux.HandleFunc("POST /v1/abuse/reports", s.abuseReport)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /readyz", s.ready)

	return s.wrap(mux)
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "err", rec)
				problem.Write(w, problem.New(500, "Internal error", "unexpected error", r.URL.Path))
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			problem.Write(w, problem.New(401, "Unauthorized", "missing bearer token", r.URL.Path))
			return
		}
		tok := strings.TrimSpace(h[7:])
		var p auth.Principal
		if auth.IsAPIKey(tok) {
			row, err := s.repo.APIKeyByHash(r.Context(), auth.HashAPIKey(tok))
			if err != nil || row.RevokedAt != nil {
				problem.Write(w, problem.New(401, "Unauthorized", "invalid API key", r.URL.Path))
				return
			}
			if row.ExpiresAt != nil && time.Now().After(*row.ExpiresAt) {
				problem.Write(w, problem.New(401, "Unauthorized", "expired API key", r.URL.Path))
				return
			}
			p = auth.Principal{APIKeyID: row.ID, WorkspaceID: row.WorkspaceID, Role: "admin", Scopes: row.Scopes}
			go s.repo.TouchAPIKey(context.Background(), row.ID)
		} else {
			uid, err := auth.ParseJWT(s.cfg.TokenKey, tok)
			if err != nil {
				problem.Write(w, problem.New(401, "Unauthorized", "invalid token", r.URL.Path))
				return
			}
			p = auth.Principal{UserID: uid}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	}
}

func (s *Server) requireWS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFrom(r)
		wsID := p.WorkspaceID
		if raw := r.PathValue("id"); raw != "" && strings.HasPrefix(r.URL.Path, "/v1/workspaces/") {
			if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
				wsID = id
			}
		}
		if wsID == 0 {
			if raw := r.Header.Get("X-Workspace-Id"); raw != "" {
				id, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					problem.Write(w, problem.New(400, "Bad request", "invalid X-Workspace-Id", r.URL.Path))
					return
				}
				wsID = id
			} else if p.UserID != 0 {
				m, err := s.repo.FirstMembership(r.Context(), p.UserID)
				if err != nil {
					problem.Write(w, problem.New(403, "Forbidden", "no workspace membership", r.URL.Path))
					return
				}
				wsID, p.Role = m.WorkspaceID, m.Role
			}
		}
		if p.UserID != 0 && p.Role == "" {
			m, err := s.repo.Membership(r.Context(), wsID, p.UserID)
			if err != nil {
				problem.Write(w, problem.New(403, "Forbidden", "not a member of this workspace", r.URL.Path))
				return
			}
			p.Role = m.Role
		}
		if wsID == 0 {
			problem.Write(w, problem.New(400, "Bad request", "workspace required", r.URL.Path))
			return
		}
		p.WorkspaceID = wsID
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.repo.Ping(r.Context()); err != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ready"))
}

// ---- auth ----

type registerReq struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Workspace  string `json:"workspace"`
	HomeRegion string `json:"home_region"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in registerReq
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	in.Email = strings.TrimSpace(in.Email)
	if in.Email == "" || len(in.Password) < 10 {
		problem.Write(w, problem.New(422, "Validation failed", "email required and password must be ≥10 characters", r.URL.Path))
		return
	}
	if in.Workspace == "" {
		in.Workspace = strings.Split(in.Email, "@")[0] + "'s workspace"
	}
	if in.HomeRegion == "" {
		in.HomeRegion = "us"
	}
	if in.HomeRegion != "us" && in.HomeRegion != "eu" {
		problem.Write(w, problem.New(422, "Validation failed", "home_region must be us or eu", r.URL.Path))
		return
	}
	hash, err := argon2id.Hash(in.Password)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "hash failed", r.URL.Path))
		return
	}
	uid, wsID, err := s.repo.CreateUserWithWorkspace(r.Context(), in.Email, hash, in.Workspace, in.HomeRegion)
	if errors.Is(err, repo.ErrConflict) {
		problem.Write(w, problem.New(409, "Conflict", "email already registered", r.URL.Path))
		return
	}
	if err != nil {
		s.log.Error("register", "err", err)
		problem.Write(w, problem.New(500, "Internal error", "could not create account", r.URL.Path))
		return
	}
	tok, err := auth.IssueJWT(s.cfg.TokenKey, uid, s.cfg.TokenTTL)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "token", r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), wsID, uid, 0, "user.register", "user", strconv.FormatInt(uid, 10))
	writeJSON(w, 201, map[string]any{"token": tok, "user_id": uid, "workspace_id": wsID})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTP     string `json:"totp"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	u, err := s.repo.GetUserByEmail(r.Context(), in.Email)
	if err != nil || u.PasswordHash == nil || !argon2id.Verify(in.Password, *u.PasswordHash) {
		problem.Write(w, problem.New(401, "Unauthorized", "invalid credentials", r.URL.Path))
		return
	}
	if u.Status != "active" {
		problem.Write(w, problem.New(403, "Forbidden", "account not active", r.URL.Path))
		return
	}
	if u.MFAEnabled {
		plain, err := auth.DecryptAESGCM(s.cfg.TokenKey, u.MFASecret)
		if err != nil || !auth.VerifyTOTP(plain, in.TOTP, time.Now()) {
			problem.Write(w, problem.New(401, "Unauthorized", "invalid TOTP", r.URL.Path))
			return
		}
	}
	tok, err := auth.IssueJWT(s.cfg.TokenKey, u.ID, s.cfg.TokenTTL)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "token", r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok, "user_id": u.ID, "mfa_required": false})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	// JWTs are short-lived; clients drop the token. A denylist can be added via Redis if needed.
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		writeJSON(w, 200, map[string]any{"api_key_id": p.APIKeyID, "workspace_id": p.WorkspaceID, "role": p.Role})
		return
	}
	u, err := s.repo.GetUser(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(404, "Not found", "user", r.URL.Path))
		return
	}
	ws, _ := s.repo.ListMemberships(r.Context(), p.UserID)
	writeJSON(w, 200, map[string]any{
		"id": u.ID, "email": u.Email, "mfa_enabled": u.MFAEnabled,
		"gpc_optout": u.GPCOptOut, "home_region": u.HomeRegion, "workspaces": ws,
	})
}

func (s *Server) totpSetup(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		problem.Write(w, problem.New(403, "Forbidden", "user session required", r.URL.Path))
		return
	}
	u, err := s.repo.GetUser(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(404, "Not found", "user", r.URL.Path))
		return
	}
	raw, b32, err := auth.NewTOTPSecret()
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "totp", r.URL.Path))
		return
	}
	enc, err := auth.EncryptAESGCM(s.cfg.TokenKey, raw)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "encrypt", r.URL.Path))
		return
	}
	if err := s.repo.SetTOTP(r.Context(), p.UserID, enc, false); err != nil {
		problem.Write(w, problem.New(500, "Internal error", "store totp", r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{
		"secret": b32,
		"otpauth_url": auth.OTPAuthURL(u.Email, b32, "Linkspire"),
	})
}

type totpEn struct {
	Code string `json:"code"`
}

func (s *Server) totpEnable(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	var in totpEn
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	u, err := s.repo.GetUser(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(404, "Not found", "user", r.URL.Path))
		return
	}
	plain, err := auth.DecryptAESGCM(s.cfg.TokenKey, u.MFASecret)
	if err != nil || !auth.VerifyTOTP(plain, in.Code, time.Now()) {
		problem.Write(w, problem.New(401, "Unauthorized", "invalid TOTP", r.URL.Path))
		return
	}
	if err := s.repo.SetTOTP(r.Context(), p.UserID, u.MFASecret, true); err != nil {
		problem.Write(w, problem.New(500, "Internal error", "enable totp", r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"mfa_enabled": true})
}

func (s *Server) oidcStart(w http.ResponseWriter, r *http.Request) {
	if s.cfg.OIDCIssuer == "" {
		problem.Write(w, problem.New(501, "Not implemented", "OIDC is not configured on this deployment", r.URL.Path))
		return
	}
	// Authorization URL is assembled by the configured issuer; the callback completes login.
	u := s.cfg.OIDCIssuer + "/authorize?client_id=" + s.cfg.OIDCClientID +
		"&redirect_uri=" + s.cfg.OIDCRedirectURL + "&response_type=code&scope=openid%20email"
	http.Redirect(w, r, u, http.StatusFound)
}

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	if s.cfg.OIDCIssuer == "" {
		problem.Write(w, problem.New(501, "Not implemented", "OIDC is not configured on this deployment", r.URL.Path))
		return
	}
	problem.Write(w, problem.New(501, "Not implemented", "OIDC token exchange requires issuer JWKS wiring in this environment", r.URL.Path))
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		writeJSON(w, 200, []repo.Membership{{WorkspaceID: p.WorkspaceID, Role: p.Role}})
		return
	}
	ms, err := s.repo.ListMemberships(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, ms)
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	ms, err := s.repo.ListWorkspaceMembers(r.Context(), p.WorkspaceID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, ms)
}

func (s *Server) putMember(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanOwner() {
		problem.Write(w, problem.New(403, "Forbidden", "owner required", r.URL.Path))
		return
	}
	uid, _ := strconv.ParseInt(r.PathValue("uid"), 10, 64)
	var in struct {
		Role string `json:"role"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	switch in.Role {
	case "owner", "admin", "member", "viewer":
	default:
		problem.Write(w, problem.New(422, "Validation failed", "invalid role", r.URL.Path))
		return
	}
	if err := s.repo.UpsertMember(r.Context(), p.WorkspaceID, uid, in.Role); err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"user_id": uid, "role": in.Role})
}

// ---- links ----

type createLinkReq struct {
	Destination  string  `json:"destination"`
	DomainID     int64   `json:"domain_id"`
	Alias        string  `json:"alias"`
	RedirectType int16   `json:"redirect_type"`
	Password     string  `json:"password"`
	ExpiresAt    *string `json:"expires_at"`
	ClickLimit   *int64  `json:"click_limit"`
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	Image        *string `json:"image"`
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanWrite() {
		problem.Write(w, problem.New(403, "Forbidden", "write role required", r.URL.Path))
		return
	}
	var in createLinkReq
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	l, err := s.mintLink(r.Context(), p, in)
	if err != nil {
		s.writeLinkErr(w, r, err)
		return
	}
	if ik := r.Header.Get("Idempotency-Key"); ik != "" {
		sum := sha256.Sum256([]byte(in.Destination + "|" + in.Alias))
		_ = s.repo.IdemPut(r.Context(), p.WorkspaceID, ik, l.Code, 201, sum[:])
	}
	writeJSON(w, 201, s.linkDTO(l))
}

func (s *Server) bulkLinks(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanWrite() {
		problem.Write(w, problem.New(403, "Forbidden", "write role required", r.URL.Path))
		return
	}
	var in struct {
		Items []createLinkReq `json:"items"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	if len(in.Items) == 0 || len(in.Items) > 100 {
		problem.Write(w, problem.New(422, "Validation failed", "items must be 1–100", r.URL.Path))
		return
	}
	out := make([]any, 0, len(in.Items))
	for _, item := range in.Items {
		l, err := s.mintLink(r.Context(), p, item)
		if err != nil {
			out = append(out, map[string]any{"error": err.Error()})
			continue
		}
		out = append(out, s.linkDTO(l))
	}
	writeJSON(w, 200, map[string]any{"data": out})
}

func (s *Server) mintLink(ctx context.Context, p auth.Principal, in createLinkReq) (repo.Link, error) {
	dest, err := urlpolicy.ValidateDestination(in.Destination)
	if err != nil {
		return repo.Link{}, err
	}
	domainID := in.DomainID
	host := s.cfg.DefaultDomainHost
	if domainID == 0 {
		domainID = s.cfg.DefaultDomainID
	} else {
		h, ok, err := s.repo.DomainOwned(ctx, p.WorkspaceID, domainID)
		if err != nil {
			return repo.Link{}, err
		}
		if !ok {
			return repo.Link{}, fmt.Errorf("domain not in workspace")
		}
		host = h
	}
	verdict := scan.FastPath(ctx, dest, s.gsb)
	if verdict == scan.Malicious {
		return repo.Link{}, errMalicious
	}
	code := in.Alias
	if code != "" {
		reserved, _ := s.repo.ReservedSlugs(ctx)
		if reserved == nil {
			reserved = map[string]struct{}{}
		}
		for _, b := range slug.BuiltinReserved {
			reserved[b] = struct{}{}
		}
		code, err = slug.ValidateAlias(code, reserved)
		if err != nil {
			return repo.Link{}, err
		}
	} else {
		code, err = s.alloc.NextCode(ctx)
		if err != nil {
			return repo.Link{}, err
		}
	}
	status := "active"
	pw := false
	if in.Password != "" {
		status = "password_protected"
		pw = true
	}
	rt := in.RedirectType
	if rt == 0 {
		rt = 302
	}
	if rt != 301 && rt != 302 && rt != 307 && rt != 308 {
		return repo.Link{}, fmt.Errorf("redirect_type must be 301, 302, 307, or 308")
	}
	var exp *time.Time
	if in.ExpiresAt != nil && *in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
		if err != nil {
			return repo.Link{}, fmt.Errorf("expires_at must be RFC3339")
		}
		exp = &t
	}
	sv := string(verdict)
	if sv == "" {
		sv = "pending"
	}
	l := repo.Link{
		DomainID: domainID, Code: code, WorkspaceID: p.WorkspaceID, CreatedBy: p.UserID,
		Destination: dest, Status: status, Verdict: sv, RedirectType: rt,
		PasswordProtected: pw, Title: in.Title, Description: in.Description, Image: in.Image,
		ExpiresAt: exp, ClickLimit: in.ClickLimit, CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.InsertLink(ctx, l); err != nil {
		return repo.Link{}, err
	}
	if pw {
		phc, err := argon2id.Hash(in.Password)
		if err != nil {
			return repo.Link{}, err
		}
		if err := s.repo.SetLinkPassword(ctx, domainID, code, phc); err != nil {
			return repo.Link{}, err
		}
	}
	s.repo.Audit(ctx, p.WorkspaceID, p.UserID, p.APIKeyID, "link.create", "link", fmt.Sprintf("%d:%s", domainID, code))
	_ = host
	s.fanout(ctx, p.WorkspaceID, "link.created", map[string]any{
		"domain_id": domainID, "code": code, "short_url": s.cfg.PublicBaseURL + "/" + code,
	})
	return l, nil
}

var errMalicious = errors.New("destination rejected as malicious")

func (s *Server) writeLinkErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errMalicious):
		problem.Write(w, problem.New(422, "Rejected", "destination failed malware/phishing checks", r.URL.Path))
	case errors.Is(err, repo.ErrConflict):
		problem.Write(w, problem.New(409, "Conflict", "code already exists on this domain", r.URL.Path))
	default:
		problem.Write(w, problem.New(422, "Validation failed", err.Error(), r.URL.Path))
	}
}

func (s *Server) linkDTO(l repo.Link) map[string]any {
	return map[string]any{
		"domain_id": l.DomainID, "code": l.Code, "destination": l.Destination,
		"status": l.Status, "scan_verdict": l.Verdict, "redirect_type": l.RedirectType,
		"password_protected": l.PasswordProtected,
		"short_url":          s.cfg.PublicBaseURL + "/" + l.Code,
		"expires_at":         l.ExpiresAt, "click_limit": l.ClickLimit,
		"created_at": l.CreatedAt,
	}
}

func (s *Server) listLinks(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	ls, err := s.repo.ListLinks(r.Context(), p.WorkspaceID, n)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	data := make([]any, 0, len(ls))
	for _, l := range ls {
		data = append(data, s.linkDTO(l))
	}
	writeJSON(w, 200, map[string]any{"data": data, "next_cursor": nil})
}

func (s *Server) getLink(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	l, err := s.repo.GetLink(r.Context(), p.WorkspaceID, did, r.PathValue("code"))
	if errors.Is(err, repo.ErrNotFound) {
		problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
		return
	}
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, s.linkDTO(l))
}

func (s *Server) patchLink(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanWrite() {
		problem.Write(w, problem.New(403, "Forbidden", "write role required", r.URL.Path))
		return
	}
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	code := r.PathValue("code")
	var in struct {
		Destination  string `json:"destination"`
		Status       string `json:"status"`
		RedirectType int16  `json:"redirect_type"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	if in.Destination != "" {
		d, err := urlpolicy.ValidateDestination(in.Destination)
		if err != nil {
			problem.Write(w, problem.New(422, "Validation failed", err.Error(), r.URL.Path))
			return
		}
		if scan.FastPath(r.Context(), d, s.gsb) == scan.Malicious {
			problem.Write(w, problem.New(422, "Rejected", "destination failed malware/phishing checks", r.URL.Path))
			return
		}
		in.Destination = d
	}
	if err := s.repo.UpdateLink(r.Context(), p.WorkspaceID, did, code, in.Destination, in.Status, in.RedirectType); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
			return
		}
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), p.WorkspaceID, p.UserID, p.APIKeyID, "link.update", "link", fmt.Sprintf("%d:%s", did, code))
	l, _ := s.repo.GetLink(r.Context(), p.WorkspaceID, did, code)
	writeJSON(w, 200, s.linkDTO(l))
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanWrite() {
		problem.Write(w, problem.New(403, "Forbidden", "write role required", r.URL.Path))
		return
	}
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	code := r.PathValue("code")
	if err := s.repo.SoftDeleteLink(r.Context(), p.WorkspaceID, did, code); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
			return
		}
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), p.WorkspaceID, p.UserID, p.APIKeyID, "link.delete", "link", fmt.Sprintf("%d:%s", did, code))
	w.WriteHeader(204)
}

func (s *Server) fanout(ctx context.Context, wsID int64, event string, payload any) {
	ids, err := s.repo.ActiveWebhookIDs(ctx, wsID, event)
	if err != nil {
		return
	}
	for _, id := range ids {
		_ = s.repo.EnqueueWebhook(ctx, id, wsID, event, payload)
	}
}
