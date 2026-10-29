// services/workers/main.go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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
