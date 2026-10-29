// services/workers/internal/qrgen/qrgen.go
package qrgen

import (
	"context"
	"fmt"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/skip2/go-qrcode"

	"github.com/linkspire/linkspire/services/workers/internal/store"
)

type Worker struct {
	Store  *store.Store
	Dir    string
	Base   string
	Log    *slog.Logger
	Public string // public URL prefix for the short link encoded in the QR
}

func (w *Worker) Tick(ctx context.Context) {
	if err := os.MkdirAll(w.Dir, 0o755); err != nil {
		w.Log.Error("qr mkdir", "err", err)
		return
	}
	rows, err := w.Store.QRPending(ctx, 20)
	if err != nil {
		w.Log.Error("qr list", "err", err)
		return
	}
	for _, q := range rows {
		content := fmt.Sprintf("%s/%s?s=qr", strings.TrimRight(w.Public, "/"), q.Code)
		path, err := w.render(q, content)
		if err != nil {
			w.Log.Warn("qr render", "id", q.ID, "err", err)
			continue
		}
		_ = w.Store.QRSetKey(ctx, q.ID, path)
	}
}

func (w *Worker) render(q store.QRRow, content string) (string, error) {
	lvl := qrcode.Medium
	switch q.EC {
	case "L":
		lvl = qrcode.Low
	case "Q":
		lvl = qrcode.High
	case "H":
		lvl = qrcode.Highest
	}
	if q.HasLogo {
		lvl = qrcode.Highest
	}
	qr, err := qrcode.New(content, lvl)
	if err != nil {
		return "", err
	}
	qr.ForegroundColor = parseHex(q.FG, color.Black)
	qr.BackgroundColor = parseHex(q.BG, color.White)
	name := fmt.Sprintf("%d-%s.%s", q.ID, q.Code, q.Format)
	full := filepath.Join(w.Dir, name)
	if q.Format == "svg" {
		return full, os.WriteFile(full, bitmapSVG(qr, q.Size), 0o644)
	}
	img := qr.Image(q.Size)
	f, err := os.Create(full)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return full, png.Encode(f, img)
}

func parseHex(s string, fallback color.Color) color.Color {
	s = strings.TrimPrefix(s, "#")
	var r, g, b uint8
	switch len(s) {
	case 6:
		var n uint32
		if _, err := fmt.Sscanf(s, "%06x", &n); err != nil {
			return fallback
		}
		r, g, b = uint8(n>>16), uint8(n>>8), uint8(n)
	default:
		return fallback
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func bitmapSVG(qr *qrcode.QRCode, size int) []byte {
	img := qr.Image(size)
	b := img.Bounds()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r < 1<<15 && g < 1<<15 && bl < 1<<15 {
				sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1"/>`, x, y))
			}
		}
	}
	sb.WriteString(`</svg>`)
	return []byte(sb.String())
}
