// services/api/main.go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/linkspire/linkspire/pkg/codec"
	"github.com/linkspire/linkspire/pkg/idgen"
	"github.com/linkspire/linkspire/pkg/scan"
	"github.com/linkspire/linkspire/services/api/internal/config"
	"github.com/linkspire/linkspire/services/api/internal/httpserver"
	"github.com/linkspire/linkspire/services/api/internal/repo"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := repo.New(ctx, cfg.PGDSN, cfg.PGMaxConns)
	if err != nil {
		return err
	}
	defer db.Close()

	gen, err := codec.NewGenerator(cfg.FeistelKey, codec.DefaultWidth, codec.DefaultRounds)
	if err != nil {
		return err
	}
	alloc := idgen.NewAllocator(db.Pool, gen, 10_000)

	gsb := &scan.HashPrefixDB{}
	if err := gsb.LoadFile(cfg.GSBFile); err != nil {
		return err
	}

	api := httpserver.New(cfg, db, alloc, gsb, log)
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      api.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}
	admin := &http.Server{Addr: cfg.MetricsAddr, Handler: promhttp.Handler()}

	errCh := make(chan error, 2)
	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		if err := admin.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal")
	case err := <-errCh:
		log.Error("server error", "err", err)
	}
	shctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shctx)
	_ = admin.Shutdown(shctx)
	return nil
}
