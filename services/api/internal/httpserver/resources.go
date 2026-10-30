// services/api/internal/httpserver/resources.go
package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/linkspire/linkspire/pkg/problem"
	"github.com/linkspire/linkspire/services/api/internal/auth"
	"github.com/linkspire/linkspire/services/api/internal/repo"
)

func (s *Server) createDomain(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	var in struct {
		Hostname string `json:"hostname"`
		Apex     bool   `json:"apex"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	host := strings.ToLower(strings.TrimSpace(in.Hostname))
	if host == "" || strings.ContainsAny(host, " /:") {
		problem.Write(w, problem.New(422, "Validation failed", "invalid hostname", r.URL.Path))
		return
	}
	var tok [16]byte
	_, _ = rand.Read(tok[:])
	token := hex.EncodeToString(tok[:])
	id, err := s.repo.InsertDomain(r.Context(), p.WorkspaceID, host, token, "cname.linkspire.io", in.Apex)
	if errors.Is(err, repo.ErrConflict) {
		problem.Write(w, problem.New(409, "Conflict", "hostname already claimed", r.URL.Path))
		return
	}
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), p.WorkspaceID, p.UserID, p.APIKeyID, "domain.create", "domain", strconv.FormatInt(id, 10))
	writeJSON(w, 201, map[string]any{
		"id": id, "hostname": host, "state": "pending",
		"cname_target": "cname.linkspire.io",
		"txt_name":     "_linkspire-verify." + host,
		"txt_value":    token,
		"apex":         in.Apex,
		"apex_note":    "Apex domains cannot CNAME (RFC 1034). Use ALIAS/ANAME, or an A-record fallback to our anycast IPs.",
	})
}

func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	ds, err := s.repo.ListDomains(r.Context(), p.WorkspaceID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"data": ds})
}

func (s *Server) getDomain(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	d, err := s.repo.GetDomain(r.Context(), p.WorkspaceID, id)
	if errors.Is(err, repo.ErrNotFound) {
		problem.Write(w, problem.New(404, "Not found", "domain", r.URL.Path))
		return
	}
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) patchDomain(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		RootRedirect *string `json:"root_redirect_url"`
		Custom404    *string `json:"custom_404_url"`
		HSTS         bool    `json:"hsts_enabled"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	if err := s.repo.UpdateDomainSettings(r.Context(), p.WorkspaceID, id, in.RootRedirect, in.Custom404, in.HSTS); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			problem.Write(w, problem.New(404, "Not found", "domain", r.URL.Path))
			return
		}
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	d, _ := s.repo.GetDomain(r.Context(), p.WorkspaceID, id)
	writeJSON(w, 200, d)
}

func (s *Server) requestQR(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanWrite() {
		problem.Write(w, problem.New(403, "Forbidden", "write role required", r.URL.Path))
		return
	}
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	code := r.PathValue("code")
	if _, err := s.repo.GetLink(r.Context(), p.WorkspaceID, did, code); err != nil {
		problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
		return
	}
	var in struct {
		Format  string `json:"format"`
		Size    int    `json:"size"`
		Margin  int    `json:"margin"`
		ECLevel string `json:"ec_level"`
		FG      string `json:"fg_color"`
		BG      string `json:"bg_color"`
		Logo    bool   `json:"logo"`
	}
	if r.ContentLength > 0 {
		_ = decode(r, &in)
	}
	if in.Format == "" {
		in.Format = "png"
	}
	if in.Format != "png" && in.Format != "svg" {
		problem.Write(w, problem.New(422, "Validation failed", "format must be png or svg", r.URL.Path))
		return
	}
	if in.Size == 0 {
		in.Size = 512
	}
	if in.Margin == 0 {
		in.Margin = 4
	}
	if in.ECLevel == "" {
		in.ECLevel = "M"
	}
	if in.Logo {
		in.ECLevel = "H"
	}
	if in.FG == "" {
		in.FG = "#000000"
	}
	if in.BG == "" {
		in.BG = "#FFFFFF"
	}
	style := strings.Join([]string{in.Format, strconv.Itoa(in.Size), in.ECLevel, in.FG, in.BG}, "|")
	id, err := s.repo.InsertQR(r.Context(), did, code, in.Format, in.ECLevel, in.Size, in.Margin, in.FG, in.BG, in.Logo, style, "")
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 202, map[string]any{"id": id, "status": "queued", "style_hash": style})
}

func (s *Server) createKey(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	var in struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	if in.Name == "" {
		in.Name = "default"
	}
	plain, prefix, hash, err := auth.NewAPIKey()
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", "keygen", r.URL.Path))
		return
	}
	id, err := s.repo.InsertAPIKey(r.Context(), p.WorkspaceID, p.UserID, in.Name, prefix, hash, in.Scopes)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), p.WorkspaceID, p.UserID, p.APIKeyID, "apikey.create", "api_key", strconv.FormatInt(id, 10))
	writeJSON(w, 201, map[string]any{"id": id, "prefix": prefix, "key": plain, "note": "store this key now; it will not be shown again"})
}

func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	ks, err := s.repo.ListAPIKeys(r.Context(), p.WorkspaceID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"data": ks})
}

func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.repo.RevokeAPIKey(r.Context(), p.WorkspaceID, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			problem.Write(w, problem.New(404, "Not found", "api key", r.URL.Path))
			return
		}
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	w.WriteHeader(204)
}

func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	var in struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	if !strings.HasPrefix(in.URL, "https://") {
		problem.Write(w, problem.New(422, "Validation failed", "webhook URL must be https", r.URL.Path))
		return
	}
	var sec [32]byte
	_, _ = rand.Read(sec[:])
	id, err := s.repo.InsertWebhook(r.Context(), p.WorkspaceID, in.URL, sec[:], in.Events)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 201, map[string]any{"id": id, "url": in.URL, "events": in.Events, "secret_hex": hex.EncodeToString(sec[:])})
}

func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	ws, err := s.repo.ListWebhooks(r.Context(), p.WorkspaceID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"data": ws})
}

func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if !p.CanAdmin() {
		problem.Write(w, problem.New(403, "Forbidden", "admin required", r.URL.Path))
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.repo.DeleteWebhook(r.Context(), p.WorkspaceID, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			problem.Write(w, problem.New(404, "Not found", "webhook", r.URL.Path))
			return
		}
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	w.WriteHeader(204)
}

func (s *Server) analytics(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	code := r.PathValue("code")
	if _, err := s.repo.GetLink(r.Context(), p.WorkspaceID, did, code); err != nil {
		problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
		return
	}
	if s.cfg.ClickHouseURL == "" {
		writeJSON(w, 200, map[string]any{
			"data":    []any{},
			"note":    "ClickHouse is not configured; aggregates unavailable. Unique visitors are per-day only (daily-rotating salt).",
			"privacy": "no raw IP or user-agent is stored",
		})
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 90
	}
	// Best-effort query; failure degrades to empty rather than blocking the API SLO.
	rows, err := s.clickhouseRollups(r.Context(), did, code, days)
	if err != nil {
		s.log.Warn("clickhouse query", "err", err)
		writeJSON(w, 200, map[string]any{
			"data":    []any{},
			"source":  "clickhouse",
			"note":    "analytics temporarily unavailable; serving empty aggregates",
			"privacy": "no raw IP or user-agent is stored",
		})
		return
	}
	writeJSON(w, 200, map[string]any{"data": rows, "source": "clickhouse"})
}

func (s *Server) createDSAR(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		problem.Write(w, problem.New(403, "Forbidden", "user session required", r.URL.Path))
		return
	}
	var in struct {
		Type   string `json:"type"`
		Regime string `json:"regime"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	switch in.Type {
	case "access", "rectification", "erasure", "restriction", "objection", "do_not_sell":
	default:
		problem.Write(w, problem.New(422, "Validation failed", "unknown request type", r.URL.Path))
		return
	}
	if in.Regime == "" {
		in.Regime = "gdpr"
	}
	sla := 30 * 24 * time.Hour
	if in.Regime == "ccpa" {
		sla = 45 * 24 * time.Hour
	}
	u, err := s.repo.GetUser(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(404, "Not found", "user", r.URL.Path))
		return
	}
	wsID := p.WorkspaceID
	if wsID == 0 {
		if m, err := s.repo.FirstMembership(r.Context(), p.UserID); err == nil {
			wsID = m.WorkspaceID
		}
	}
	id, err := s.repo.InsertDSAR(r.Context(), wsID, p.UserID, u.Email, in.Type, in.Regime, time.Now().Add(sla))
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 202, map[string]any{"id": id, "status": "received", "sla_days": int(sla.Hours() / 24)})
}

func (s *Server) listDSAR(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	ds, err := s.repo.ListDSAR(r.Context(), p.UserID)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{"data": ds})
}

func (s *Server) getDSAR(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	d, err := s.repo.GetDSAR(r.Context(), id, p.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		problem.Write(w, problem.New(404, "Not found", "request", r.URL.Path))
		return
	}
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) doNotSell(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	if p.UserID == 0 {
		problem.Write(w, problem.New(403, "Forbidden", "user session required", r.URL.Path))
		return
	}
	if err := s.repo.SetGPC(r.Context(), p.UserID, true); err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	writeJSON(w, 200, map[string]any{
		"do_not_sell": true,
		"statement":   "Linkspire does not sell or share personal information. This flag records your CCPA/CPRA preference.",
	})
}

func (s *Server) tlsAsk(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AskToken != "" && r.Header.Get("X-Ask-Token") != s.cfg.AskToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	host := strings.ToLower(r.URL.Query().Get("domain"))
	if host == "" {
		http.Error(w, "missing domain", http.StatusBadRequest)
		return
	}
	_, _, state, err := s.repo.DomainByHost(r.Context(), host)
	if err != nil || state != "active" {
		http.Error(w, "unauthorized host", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) abuseReport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ShortURL string `json:"short_url"`
		Reason   string `json:"reason"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	if _, err := s.repo.InsertAbuseReport(r.Context(), in.ShortURL, in.Reason, nil); err != nil {
		s.log.Error("abuse insert", "err", err)
		problem.Write(w, problem.New(500, "Internal error", "could not record report", r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), 0, 0, 0, "abuse.report", "link", in.ShortURL)
	writeJSON(w, 202, map[string]any{"status": "queued", "note": "report entered the admin review queue"})
}

func (s *Server) listQR(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	code := r.PathValue("code")
	if _, err := s.repo.GetLink(r.Context(), p.WorkspaceID, did, code); err != nil {
		problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
		return
	}
	rows, err := s.repo.ListQR(r.Context(), did, code)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	base := strings.TrimRight(s.cfg.QRPublicBase, "/")
	data := make([]any, 0, len(rows))
	for _, q := range rows {
		status := "pending"
		url := ""
		if q.ObjectKey != nil && strings.TrimSpace(*q.ObjectKey) != "" {
			status = "ready"
			if base != "" {
				url = base + "/" + strings.TrimPrefix(*q.ObjectKey, "/")
			}
		}
		data = append(data, map[string]any{
			"id": q.ID, "format": q.Format, "ec_level": q.ECLevel, "size": q.Size,
			"status": status, "url": url, "created_at": q.CreatedAt,
		})
	}
	writeJSON(w, 200, map[string]any{"data": data})
}

func (s *Server) downloadQRFile(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	did, _ := strconv.ParseInt(r.PathValue("domainId"), 10, 64)
	code := r.PathValue("code")
	qid, _ := strconv.ParseInt(r.PathValue("qrId"), 10, 64)
	if _, err := s.repo.GetLink(r.Context(), p.WorkspaceID, did, code); err != nil {
		problem.Write(w, problem.New(404, "Not found", "link", r.URL.Path))
		return
	}
	q, err := s.repo.GetQR(r.Context(), did, code, qid)
	if err != nil || q.ObjectKey == nil || strings.TrimSpace(*q.ObjectKey) == "" {
		problem.Write(w, problem.New(404, "Not found", "qr asset", r.URL.Path))
		return
	}
	if s.cfg.QRAssetDir == "" {
		problem.Write(w, problem.New(503, "Unavailable", "QR asset storage is not configured on the API", r.URL.Path))
		return
	}
	key := filepath.Base(strings.ReplaceAll(*q.ObjectKey, "\\", "/"))
	path := filepath.Join(s.cfg.QRAssetDir, key)
	cleanRoot := filepath.Clean(s.cfg.QRAssetDir)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) && cleanPath != cleanRoot {
		problem.Write(w, problem.New(500, "Internal error", "invalid qr path", r.URL.Path))
		return
	}
	f, err := os.Open(cleanPath)
	if err != nil {
		problem.Write(w, problem.New(404, "Not found", "qr file missing on disk", r.URL.Path))
		return
	}
	defer f.Close()
	ct := "image/png"
	if q.Format == "svg" {
		ct = "image/svg+xml"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, code, q.Format))
	http.ServeContent(w, r, key, q.CreatedAt, f)
}
