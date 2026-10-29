// services/workers/main.go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/linkspire/linkspire/pkg/scan"
	"github.com/linkspire/linkspire/services/workers/internal/config"
	"github.com/linkspire/linkspire/services/workers/internal/dsar"
	"github.com/linkspire/linkspire/services/workers/internal/qrgen"
	"github.com/linkspire/linkspire/services/workers/internal/retention"
	"github.com/linkspire/linkspire/services/workers/internal/scanner"
	"github.com/linkspire/linkspire/services/workers/internal/scraper"
	"github.com/linkspire/linkspire/services/workers/internal/store"
	"github.com/linkspire/linkspire/services/workers/internal/verifier"
	"github.com/linkspire/linkspire/services/workers/internal/webhooks"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.PGDSN, cfg.PGMaxConns)
	if err != nil {
		return err
	}
	defer st.Close()

	// Observability server: liveness, readiness (DB ping), and Prometheus metrics.
	// Workers have no inbound traffic, so this is the only HTTP surface and is used
	// by Kubernetes probes.
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	adminMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		pctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := st.Pool.Ping(pctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready: db"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	adminMux.Handle("/metrics", promhttp.Handler())
	adminSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: adminMux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("admin server", "err", err)
		}
	}()
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = adminSrv.Shutdown(sctx)
	}()

	gsb := &scan.HashPrefixDB{}
	_ = gsb.LoadFile(cfg.GSBFile)

	scanPipe := &scanner.Pipeline{
		Store: st,
		Log:   log,
		Scanners: []scan.Scanner{
			scan.Heuristics{},
			gsb,
			scanner.NewURLHaus(cfg.URLHausURL, log),
		},
	}
	scrape := scraper.New(st, log)
	ver := &verifier.Worker{Store: st, Log: log}
	qr := &qrgen.Worker{Store: st, Dir: cfg.AssetDir, Base: cfg.PublicAssetBase, Public: "https://lnk.linkspire.io", Log: log}
	ret := &retention.Worker{Store: st, Log: log}
	ds := &dsar.Worker{Store: st, Log: log}
	wh := webhooks.New(st, log)

	log.Info("workers started", "role", cfg.Role)
	tick := time.NewTicker(cfg.PollInterval)
	defer tick.Stop()
	runRole := func() {
		switch cfg.Role {
		case "scanner":
			scanPipe.Tick(ctx)
		case "scraper":
			scrape.Tick(ctx)
		case "verifier":
			ver.Tick(ctx)
		case "qr":
			qr.Tick(ctx)
		case "retention":
			ret.Tick(ctx)
		case "dsar":
			ds.Tick(ctx)
		case "webhooks":
			wh.Tick(ctx)
		default:
			scanPipe.Tick(ctx)
			scrape.Tick(ctx)
			ver.Tick(ctx)
			qr.Tick(ctx)
			ds.Tick(ctx)
			wh.Tick(ctx)
			// Retention is slower; run every ~10 ticks via wall clock minute.
			if time.Now().Minute()%10 == 0 {
				ret.Tick(ctx)
			}
		}
	}
	runRole()
	for {
		select {
		case <-ctx.Done():
			log.Info("workers shutdown")
			return nil
		case <-tick.C:
			runRole()
		}
	}
}
