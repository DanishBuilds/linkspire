// services/workers/internal/dsar/dsar.go
package dsar

import (
	"context"
	"log/slog"

	"github.com/linkspire/linkspire/services/workers/internal/store"
)

type Worker struct {
	Store *store.Store
	Log   *slog.Logger
}

func (w *Worker) Tick(ctx context.Context) {
	rows, err := w.Store.OpenDSARs(ctx, 20)
	if err != nil {
		w.Log.Error("dsar list", "err", err)
		return
	}
	for _, d := range rows {
		var procErr error
		switch d.Type {
		case "erasure":
			procErr = w.Store.EraseUser(ctx, d.UserID)
		case "do_not_sell", "objection":
			procErr = w.Store.SetGPC(ctx, d.UserID, true)
		case "restriction":
			procErr = w.Store.RestrictUser(ctx, d.UserID)
		case "access", "rectification":
			// Access: export is produced by the API reading live tables; mark complete.
			// Rectification of email is performed via the profile endpoint; this flags the ticket done.
			procErr = nil
		}
		if procErr != nil {
			w.Log.Error("dsar process", "id", d.ID, "err", procErr)
			continue
		}
		if err := w.Store.CompleteDSAR(ctx, d.ID); err != nil {
			w.Log.Error("dsar complete", "id", d.ID, "err", err)
		}
	}
}
