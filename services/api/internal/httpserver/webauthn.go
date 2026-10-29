// services/api/internal/httpserver/webauthn.go
// WebAuthn / passkey registration and login using github.com/go-webauthn/webauthn.
// Ceremony state is kept in a short-lived AES-GCM encrypted cookie (no server
// session store). Requires WEBAUTHN_RP_ID and WEBAUTHN_RP_ORIGIN to be set;
// otherwise the endpoints return 501.
package httpserver

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/linkspire/linkspire/pkg/problem"
	"github.com/linkspire/linkspire/services/api/internal/auth"
)

// waUser adapts a Linkspire user to the webauthn.User interface.
type waUser struct {
	id    int64
	name  string
	creds []webauthn.Credential
}

func (u *waUser) WebAuthnID() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(u.id))
	return b
}
func (u *waUser) WebAuthnName() string                       { return u.name }
func (u *waUser) WebAuthnDisplayName() string                { return u.name }
func (u *waUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }
func (u *waUser) WebAuthnIcon() string                       { return "" }

func (s *Server) waUserFor(ctx context.Context, userID int64, email string) (*waUser, error) {
	rows, err := s.repo.ListWebAuthnByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, c := range rows {
		creds = append(creds, webauthn.Credential{
			ID:            c.CredentialID,
			PublicKey:     c.PublicKey,
			Authenticator: webauthn.Authenticator{SignCount: uint32(c.SignCount)},
		})
	}
	return &waUser{id: userID, name: email, creds: creds}, nil
}

// waEnvelope is what we store in the encrypted ceremony cookie.
type waEnvelope struct {
	UserID  int64                `json:"u"`
	Session webauthn.SessionData `json:"s"`
}

func (s *Server) waStoreSession(w http.ResponseWriter, env waEnvelope) error {
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	enc, err := auth.EncryptAESGCM(s.cfg.TokenKey, raw)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: "wa_session", Value: base64.RawURLEncoding.EncodeToString(enc), Path: "/",
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: 300,
	})
	return nil
}

func (s *Server) waReadSession(r *http.Request) (*waEnvelope, error) {
	c, err := r.Cookie("wa_session")
	if err != nil {
		return nil, err
	}
	blob, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return nil, err
	}
	raw, err := auth.DecryptAESGCM(s.cfg.TokenKey, blob)
	if err != nil {
		return nil, err
	}
	var env waEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func (s *Server) waAvailable(w http.ResponseWriter, r *http.Request) bool {
	if s.wa == nil {
		problem.Write(w, problem.New(501, "Not implemented", "WebAuthn is not configured on this deployment", r.URL.Path))
		return false
	}
	return true
}

// ---- registration (authenticated) ----

func (s *Server) webauthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if !s.waAvailable(w, r) {
		return
	}
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
	wu, err := s.waUserFor(r.Context(), p.UserID, u.Email)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	options, sd, err := s.wa.BeginRegistration(wu)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	if err := s.waStoreSession(w, waEnvelope{UserID: p.UserID, Session: *sd}); err != nil {
		problem.Write(w, problem.New(500, "Internal error", "session", r.URL.Path))
		return
	}
	writeJSON(w, 200, options)
}

func (s *Server) webauthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if !s.waAvailable(w, r) {
		return
	}
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		problem.Write(w, problem.New(403, "Forbidden", "user session required", r.URL.Path))
		return
	}
	env, err := s.waReadSession(r)
	if err != nil || env.UserID != p.UserID {
		problem.Write(w, problem.New(400, "Bad request", "missing or invalid ceremony session", r.URL.Path))
		return
	}
	u, err := s.repo.GetUser(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(404, "Not found", "user", r.URL.Path))
		return
	}
	wu, err := s.waUserFor(r.Context(), p.UserID, u.Email)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	cred, err := s.wa.FinishRegistration(wu, env.Session, r)
	if err != nil {
		problem.Write(w, problem.New(400, "Bad request", "registration failed: "+err.Error(), r.URL.Path))
		return
	}
	label := r.URL.Query().Get("label")
	if label == "" {
		label = "Passkey"
	}
	if err := s.repo.InsertWebAuthnCred(r.Context(), p.UserID, cred.ID, cred.PublicKey, int64(cred.Authenticator.SignCount), label); err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	clearShortCookie(w, "wa_session")
	s.repo.Audit(r.Context(), 0, p.UserID, 0, "webauthn.register", "user", strconv.FormatInt(p.UserID, 10))
	writeJSON(w, 201, map[string]any{"status": "registered", "label": label})
}

// ---- login (unauthenticated) ----

func (s *Server) webauthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	if !s.waAvailable(w, r) {
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	u, err := s.repo.GetUserByEmail(r.Context(), in.Email)
	if err != nil {
		problem.Write(w, problem.New(401, "Unauthorized", "no passkey for this account", r.URL.Path))
		return
	}
	wu, err := s.waUserFor(r.Context(), u.ID, u.Email)
	if err != nil || len(wu.creds) == 0 {
		problem.Write(w, problem.New(401, "Unauthorized", "no passkey for this account", r.URL.Path))
		return
	}
	options, sd, err := s.wa.BeginLogin(wu)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	if err := s.waStoreSession(w, waEnvelope{UserID: u.ID, Session: *sd}); err != nil {
		problem.Write(w, problem.New(500, "Internal error", "session", r.URL.Path))
		return
	}
	writeJSON(w, 200, options)
}

func (s *Server) webauthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	if !s.waAvailable(w, r) {
		return
	}
	env, err := s.waReadSession(r)
	if err != nil {
		problem.Write(w, problem.New(400, "Bad request", "missing or invalid ceremony session", r.URL.Path))
		return
	}
	u, err := s.repo.GetUser(r.Context(), env.UserID)
	if err != nil {
		problem.Write(w, problem.New(401, "Unauthorized", "unknown account", r.URL.Path))
		return
	}
	wu, err := s.waUserFor(r.Context(), env.UserID, u.Email)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	cred, err := s.wa.FinishLogin(wu, env.Session, r)
	if err != nil {
		problem.Write(w, problem.New(401, "Unauthorized", "passkey login failed: "+err.Error(), r.URL.Path))
		return
	}
	_ = s.repo.UpdateWebAuthnSignCount(r.Context(), cred.ID, int64(cred.Authenticator.SignCount))
	clearShortCookie(w, "wa_session")
	tok, err := auth.IssueJWT(s.cfg.TokenKey, u.ID, s.cfg.TokenTTL)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "token", r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok, "user_id": u.ID})
}

// ---- credential management (authenticated) ----

func (s *Server) webauthnList(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		problem.Write(w, problem.New(403, "Forbidden", "user session required", r.URL.Path))
		return
	}
	rows, err := s.repo.ListWebAuthnByUser(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	data := make([]any, 0, len(rows))
	for _, c := range rows {
		data = append(data, map[string]any{"id": c.ID, "label": c.Label})
	}
	writeJSON(w, 200, map[string]any{"data": data})
}

func (s *Server) webauthnDelete(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.repo.DeleteWebAuthnCred(r.Context(), p.UserID, id); err != nil {
		problem.Write(w, problem.New(404, "Not found", "passkey", r.URL.Path))
		return
	}
	w.WriteHeader(204)
}
