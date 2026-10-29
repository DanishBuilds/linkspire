// services/api/internal/httpserver/admin.go
// Platform-staff abuse review queue. Gated by users.is_staff (not workspace RBAC).
package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/linkspire/linkspire/pkg/problem"
	"github.com/linkspire/linkspire/services/api/internal/repo"
)

// requireStaff allows only authenticated platform staff (not API keys).
func (s *Server) requireStaff(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFrom(r)
		if p.UserID == 0 {
			problem.Write(w, problem.New(403, "Forbidden", "staff session required", r.URL.Path))
			return
		}
		staff, err := s.repo.IsStaff(r.Context(), p.UserID)
		if err != nil || !staff {
			problem.Write(w, problem.New(403, "Forbidden", "platform staff only", r.URL.Path))
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (s *Server) adminListAbuse(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	reports, err := s.repo.ListAbuseReports(r.Context(), status, limit)
	if err != nil {
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	data := make([]any, 0, len(reports))
	for _, a := range reports {
		data = append(data, map[string]any{
			"id": a.ID, "short_url": a.ShortURL, "reason": a.Reason, "status": a.Status,
			"reporter_email": a.ReporterEmail, "created_at": a.CreatedAt, "resolved_at": a.ResolvedAt,
		})
	}
	writeJSON(w, 200, map[string]any{"data": data})
}

func (s *Server) adminResolveAbuse(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Status string `json:"status"`
	}
	if err := decode(r, &in); err != nil {
		problem.Write(w, problem.New(400, "Bad request", err.Error(), r.URL.Path))
		return
	}
	switch in.Status {
	case "reviewing", "actioned", "dismissed":
	default:
		problem.Write(w, problem.New(422, "Validation failed", "status must be reviewing, actioned, or dismissed", r.URL.Path))
		return
	}
	if err := s.repo.ResolveAbuseReport(r.Context(), id, p.UserID, in.Status); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			problem.Write(w, problem.New(404, "Not found", "abuse report", r.URL.Path))
			return
		}
		problem.Write(w, problem.New(500, "Internal error", err.Error(), r.URL.Path))
		return
	}
	s.repo.Audit(r.Context(), 0, p.UserID, 0, "abuse.resolve", "abuse_report", strconv.FormatInt(id, 10))
	writeJSON(w, 200, map[string]any{"id": id, "status": in.Status})
}
