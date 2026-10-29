// services/workers/internal/retention/retention.go
package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/linkspire/linkspire/services/workers/internal/store"
)

type Worker struct {
	Store *store.Store
	Log   *slog.Logger
}

func (w *Worker) Tick(ctx context.Context) {
	pol, err := w.Store.RetentionPolicies(ctx)
	if err != nil {
		w.Log.Error("retention policies", "err", err)
		return
	}
	days := pol["deleted_account_data"]
	if days <= 0 {
		days = 30
	}
	n, err := w.Store.PurgeDeletedAccounts(ctx, time.Duration(days)*24*time.Hour)
	if err != nil {
		w.Log.Error("purge deleted", "err", err)
		return
	}
	w.Log.Info("retention report",
		"policies", pol,
		"purged_deleted_accounts", n,
		"note", "raw click events and security logs are dropped by store TTLs (ClickHouse / CRDB row TTL)",
	)
}
