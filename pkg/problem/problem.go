// pkg/problem/problem.go
// RFC 9457 problem+json helpers for the public API error envelope.
package problem

import (
	"encoding/json"
	"net/http"
)

// Problem is an RFC 9457 problem details object.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func (p Problem) Error() string {
	if p.Detail != "" {
		return p.Detail
	}
	return p.Title
}

// Write serializes p as application/problem+json.
func Write(w http.ResponseWriter, p Problem) {
	if p.Type == "" {
		p.Type = "about:blank"
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func New(status int, title, detail, instance string) Problem {
	return Problem{
		Type:     "https://linkspire.io/problems/" + slug(title),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	}
}

func slug(s string) string {
	b := make([]byte, 0, len(s))
	prevDash := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b = append(b, c+32)
			prevDash = false
		case c >= 'a' && c <= 'z' || c >= '0' && c <= '9':
			b = append(b, c)
			prevDash = false
		default:
			if !prevDash && len(b) > 0 {
				b = append(b, '-')
				prevDash = true
			}
		}
	}
	if len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return "error"
	}
	return string(b)
}
