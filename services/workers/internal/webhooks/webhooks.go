// services/workers/internal/webhooks/webhooks.go
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/linkspire/linkspire/pkg/ssrfguard"
	"github.com/linkspire/linkspire/services/workers/internal/store"
)

type Worker struct {
	Store  *store.Store
	Client *http.Client
	Log    *slog.Logger
}

func New(st *store.Store, log *slog.Logger) *Worker {
	return &Worker{Store: st, Client: ssrfguard.NewClient(ssrfguard.DefaultConfig()), Log: log}
}

func (w *Worker) Tick(ctx context.Context) {
	ds, err := w.Store.DueDeliveries(ctx, 20)
	if err != nil {
		w.Log.Error("webhook list", "err", err)
		return
	}
	for _, d := range ds {
		if err := w.deliver(ctx, d); err != nil {
			_ = w.Store.MarkDeliveryFail(ctx, d.ID, d.Attempts+1, err.Error())
			continue
		}
		_ = w.Store.MarkDelivered(ctx, d.ID)
	}
}

func (w *Worker) deliver(ctx context.Context, d store.Delivery) error {
	mac := hmac.New(sha256.New, d.Secret)
	mac.Write(d.Payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.URL, bytes.NewReader(d.Payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Linkspire-Signature", "sha256="+sig)
	req.Header.Set("User-Agent", "Linkspire-Webhooks/1.0")
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req = req.WithContext(cctx)
	resp, err := w.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errStatus(resp.StatusCode)
	}
	return nil
}

type statusErr int

func (e statusErr) Error() string { return http.StatusText(int(e)) }
func errStatus(c int) error       { return statusErr(c) }
