// services/api/internal/httpserver/oidc.go
// Generic OIDC Authorization Code login: issuer discovery, code exchange, and
// RS256 id_token verification against the issuer JWKS (stdlib only, no deps).
// On success the browser is redirected to OIDC_POST_LOGIN_URL with the Linkspire
// JWT in the fragment, where the dashboard exchanges it for a session cookie.
package httpserver

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linkspire/linkspire/pkg/argon2id"
	"github.com/linkspire/linkspire/pkg/problem"
	"github.com/linkspire/linkspire/services/api/internal/auth"
)

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

func oidcClient() *http.Client { return &http.Client{Timeout: 8 * time.Second} }

func (s *Server) oidcDiscover(ctx context.Context) (*oidcDiscovery, error) {
	u := strings.TrimRight(s.cfg.OIDCIssuer, "/") + "/.well-known/openid-configuration"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := oidcClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("discovery returned %d", resp.StatusCode)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" || d.JWKSURI == "" {
		return nil, errors.New("incomplete OIDC discovery document")
	}
	return &d, nil
}

func (s *Server) oidcStart(w http.ResponseWriter, r *http.Request) {
	if s.cfg.OIDCIssuer == "" || s.cfg.OIDCClientID == "" {
		problem.Write(w, problem.New(501, "Not implemented", "OIDC is not configured on this deployment", r.URL.Path))
		return
	}
	disc, err := s.oidcDiscover(r.Context())
	if err != nil {
		problem.Write(w, problem.New(502, "Bad gateway", "OIDC discovery failed: "+err.Error(), r.URL.Path))
		return
	}
	state := randToken()
	nonce := randToken()
	setShortCookie(w, "oidc_state", state, s.cfg.CookieSecure)
	setShortCookie(w, "oidc_nonce", nonce, s.cfg.CookieSecure)

	q := url.Values{}
	q.Set("client_id", s.cfg.OIDCClientID)
	q.Set("redirect_uri", s.cfg.OIDCRedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("nonce", nonce)
	http.Redirect(w, r, disc.AuthorizationEndpoint+"?"+q.Encode(), http.StatusFound)
}

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	if s.cfg.OIDCIssuer == "" {
		problem.Write(w, problem.New(501, "Not implemented", "OIDC is not configured", r.URL.Path))
		return
	}
	// CSRF: the state in the query must match the cookie we set.
	stateCookie, _ := r.Cookie("oidc_state")
	if stateCookie == nil || r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != stateCookie.Value {
		problem.Write(w, problem.New(400, "Bad request", "invalid OIDC state", r.URL.Path))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		problem.Write(w, problem.New(400, "Bad request", "missing authorization code", r.URL.Path))
		return
	}
	disc, err := s.oidcDiscover(r.Context())
	if err != nil {
		problem.Write(w, problem.New(502, "Bad gateway", "OIDC discovery failed", r.URL.Path))
		return
	}

	idToken, err := s.oidcExchange(r.Context(), disc.TokenEndpoint, code)
	if err != nil {
		problem.Write(w, problem.New(502, "Bad gateway", "token exchange failed: "+err.Error(), r.URL.Path))
		return
	}

	claims, err := s.oidcVerify(r.Context(), disc, idToken)
	if err != nil {
		problem.Write(w, problem.New(401, "Unauthorized", "id_token verification failed: "+err.Error(), r.URL.Path))
		return
	}
	nonceCookie, _ := r.Cookie("oidc_nonce")
	if nonceCookie == nil || claims.Nonce == "" || claims.Nonce != nonceCookie.Value {
		problem.Write(w, problem.New(401, "Unauthorized", "invalid OIDC nonce", r.URL.Path))
		return
	}
	if claims.Email == "" {
		problem.Write(w, problem.New(401, "Unauthorized", "id_token has no email claim", r.URL.Path))
		return
	}

	// First-login provisioning: an unusable password hash keeps the account SSO-only.
	unusable, err := argon2id.Hash(randToken())
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "provision", r.URL.Path))
		return
	}
	wsName := strings.Split(claims.Email, "@")[0] + "'s workspace"
	uid, err := s.repo.FindOrCreateOIDCUser(r.Context(), claims.Email, unusable, wsName, "us")
	if err != nil {
		s.log.Error("oidc provision", "err", err)
		problem.Write(w, problem.New(500, "Internal error", "could not provision account", r.URL.Path))
		return
	}

	tok, err := auth.IssueJWT(s.cfg.TokenKey, uid, s.cfg.TokenTTL)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "token", r.URL.Path))
		return
	}
	// Clear the single-use OIDC cookies.
	clearShortCookie(w, "oidc_state")
	clearShortCookie(w, "oidc_nonce")

	dest := s.cfg.OIDCPostLoginURL
	if dest == "" {
		dest = "/"
	}
	// Token in the fragment: never sent to a server, read by the dashboard client.
	http.Redirect(w, r, dest+"#token="+url.QueryEscape(tok), http.StatusFound)
}

func (s *Server) oidcExchange(ctx context.Context, tokenEndpoint, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.cfg.OIDCRedirectURL)
	form.Set("client_id", s.cfg.OIDCClientID)
	form.Set("client_secret", s.cfg.OIDCClientSecret)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := oidcClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.IDToken == "" {
		return "", fmt.Errorf("no id_token (%s)", out.Error)
	}
	return out.IDToken, nil
}

type oidcClaims struct {
	Iss   string `json:"iss"`
	Aud   any    `json:"aud"`
	Exp   int64  `json:"exp"`
	Email string `json:"email"`
	Nonce string `json:"nonce"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (s *Server) oidcVerify(ctx context.Context, disc *oidcDiscovery, idToken string) (*oidcClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed id_token")
	}
	var hdr struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return nil, err
	}
	if hdr.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported alg %q (only RS256)", hdr.Alg)
	}
	pub, err := s.oidcKey(ctx, disc.JWKSURI, hdr.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return nil, errors.New("signature invalid")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var c oidcClaims
	if err := json.Unmarshal(pb, &c); err != nil {
		return nil, err
	}
	if c.Iss != disc.Issuer && c.Iss != strings.TrimRight(s.cfg.OIDCIssuer, "/") {
		return nil, errors.New("issuer mismatch")
	}
	if !audContains(c.Aud, s.cfg.OIDCClientID) {
		return nil, errors.New("audience mismatch")
	}
	if time.Now().Unix() >= c.Exp {
		return nil, errors.New("id_token expired")
	}
	return &c, nil
}

func (s *Server) oidcKey(ctx context.Context, jwksURI, kid string) (*rsa.PublicKey, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	resp, err := oidcClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, err
	}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || (kid != "" && k.Kid != kid) {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eb {
			e = e<<8 | int(b)
		}
		if e == 0 {
			e = 65537
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}, nil
	}
	return nil, errors.New("no matching JWKS key")
}

func audContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

func randToken() string {
	var b [24]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func setShortCookie(w http.ResponseWriter, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", HttpOnly: true, Secure: secure,
		SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
}

func clearShortCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1})
}
