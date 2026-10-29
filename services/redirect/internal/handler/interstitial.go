// services/redirect/internal/handler/interstitial.go
package handler

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// templates holds the parsed interstitial templates. html/template auto-escapes all data,
// which is the primary XSS defense for the destination string we render in the warning page.
var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type warningData struct {
	Destination string
}

type passwordData struct {
	Code      string
	CSRFToken string
	Error     string
}
