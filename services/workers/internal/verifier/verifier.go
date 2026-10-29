// services/workers/internal/verifier/verifier.go
package verifier

import (
	"context"
	"log/slog"
	"net"
	"strings"

	"github.com/linkspire/linkspire/services/workers/internal/store"
)

type Worker struct {
	Store *store.Store
	Log   *slog.Logger
}

func (w *Worker) Tick(ctx context.Context) {
	ds, err := w.Store.DomainsDue(ctx, 20)
	if err != nil {
		w.Log.Error("domain list", "err", err)
		return
	}
	for _, d := range ds {
		okTXT, okCNAME := checkDNS(d)
		attempts := d.Attempts + 1
		switch {
		case okTXT && okCNAME && d.State == "pending":
			_ = w.Store.UpdateDomainCheck(ctx, d.ID, "verified", true, attempts)
			w.Log.Info("domain verified", "host", d.Hostname)
		case okTXT && okCNAME && d.State == "verified":
			_ = w.Store.UpdateDomainCheck(ctx, d.ID, "active", true, attempts)
			w.Log.Info("domain active", "host", d.Hostname)
		default:
			_ = w.Store.UpdateDomainCheck(ctx, d.ID, d.State, false, attempts)
		}
	}
}

func checkDNS(d store.DomainRow) (txtOK, cnameOK bool) {
	name := "_linkspire-verify." + d.Hostname
	txts, err := net.LookupTXT(name)
	if err == nil {
		for _, t := range txts {
			if t == d.Token {
				txtOK = true
				break
			}
		}
	}
	cname, err := net.LookupCNAME(d.Hostname)
	if err == nil {
		cnameOK = strings.EqualFold(strings.TrimSuffix(cname, "."), strings.TrimSuffix(d.CNAME, ".")) ||
			strings.HasSuffix(strings.ToLower(strings.TrimSuffix(cname, ".")), strings.ToLower(strings.TrimSuffix(d.CNAME, ".")))
	}
	return
}
