// services/redirect/main.go
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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/linkspire/linkspire/services/redirect/internal/cache"
	"github.com/linkspire/linkspire/services/redirect/internal/config"
	"github.com/linkspire/linkspire/services/redirect/internal/events"
	"github.com/linkspire/linkspire/services/redirect/internal/handler"
	"github.com/linkspire/linkspire/services/redirect/internal/observability"
	"github.com/linkspire/linkspire/services/redirect/internal/privacy"
	"github.com/linkspire/linkspire/services/redirect/internal/repo"
	"github.com/linkspire/linkspire/services/redirect/internal/resolver"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger.Info("config loaded", "env", cfg.Env, "http", cfg.HTTPAddr)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Observability.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics := observability.NewMetrics(reg)

	traceShutdown, err := observability.InitTracing(rootCtx, cfg.TracingEnabled, cfg.OTLPEndpoint, cfg.ServiceName)
	if err != nil {
		return err
	}

	// Dependencies (each optional / gracefully degrading).
	var (
		rd          *cache.Redis
		pgRepo      *repo.Repo
		emitter     events.Emitter = events.Nop{}
		bloom       resolver.ExistenceFilter
		l2          resolver.L2Cache
		saltProv    privacy.SaltProvider = privacy.StaticSalt(cfg.StaticVisitorSalt)
		limiter     handler.AttemptLimiter
		pwRepo      handler.PasswordRepo
		domainRepo  handler.DomainRepo
		readyChecks []handler.NamedCheck
	)

	if cfg.RedisEnabled {
		rd, err = cache.NewRedis(rootCtx, cfg.RedisAddrs, cfg.RedisPassword)
		if err != nil {
			return err
		}
		defer rd.Close()
		l2 = rd
		limiter = handler.NewRedisLimiter(rd, cfg.PWMaxAttempts, cfg.PWLockout, cfg.PWLockout)
		// Rotating visitor salt backed by Redis; refreshed hourly in memory.
		rs := privacy.NewRotatingSalt(rd.Salt, cfg.StaticVisitorSalt)
		rs.Start(rootCtx, time.Hour)
		saltProv = rs
		logger.Info("redis enabled", "addrs", cfg.RedisAddrs)
	}

	if cfg.BloomEnabled {
		if cfg.BloomUseRedis && rd != nil {
			bloom = cache.NewRedisBloom(rd, cfg.BloomRedisKey)
			logger.Info("bloom: using shared RedisBloom", "key", cfg.BloomRedisKey)
		} else {
			bloom = cache.NewLocalBloom(cfg.BloomCapacity, cfg.BloomFP)
			logger.Warn("bloom: using in-process filter; it MUST be pre-populated with all codes or valid links will be rejected")
		}
	}

	if cfg.PGEnabled {
		pgRepo, err = repo.NewRepo(rootCtx, cfg.PGDSN, cfg.PGMaxConns)
		if err != nil {
			return err
		}
		defer pgRepo.Close()
		pwRepo = pgRepo
		domainRepo = pgRepo
		readyChecks = append(readyChecks, handler.NamedCheck{Name: "postgres", Check: pgRepo})
		logger.Info("postgres enabled")
	}

	if cfg.KafkaEnabled {
		emitter = events.NewKafka(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.EventBufferSize, metrics)
		logger.Info("kafka enabled", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaTopic)
	}

	// Resolve chain.
	l1 := cache.NewL1(cfg.L1Size, cfg.L1TTL)
	res := resolver.New(resolver.Options{
		L1:           l1,
		Bloom:        bloom,
		L2:           l2,
		Repo:         repoOrNil(pgRepo),
		RecordTTL:    cfg.RedisTTL,
		RecordJitter: cfg.RedisTTLJitter,
		NegativeTTL:  cfg.NegativeTTL,
		Metrics:      metrics,
	})

	hosts := handler.NewHostResolver(domainRepo, cfg.DefaultDomainID, 5*time.Minute)

	h := handler.New(handler.Deps{
		Resolver:             res,
		Hosts:                hosts,
		Emitter:              emitter,
		Geo:                  privacy.StubGeo{},
		Salt:                 saltProv,
		PwRepo:               pwRepo,
		Limiter:              limiter,
		Metrics:              metrics,
		DefaultRedirectType:  cfg.DefaultRedirectType,
		PermanentCacheMaxAge: cfg.PermanentCacheMaxAge,
		ResolveTimeout:       cfg.ResolveTimeout,
		CookieSigningKey:     cfg.CookieSigningKey,
		CookieTTL:            cfg.CookieTTL,
		SecureCookies:        cfg.Env == "prod",
		Argon2Memory:         cfg.Argon2Memory,
		Argon2Time:           cfg.Argon2Time,
		Argon2Threads:        cfg.Argon2Threads,
		PWMaxAttempts:        cfg.PWMaxAttempts,
		PWLockout:            cfg.PWLockout,
	})

	// HTTP servers: public redirect + private metrics/health.
	redirectSrv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      h.Routes(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	health := handler.NewHealth(readyChecks...)
	adminMux := http.NewServeMux()
	adminMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	adminMux.HandleFunc("/healthz", health.Liveness)
	adminMux.HandleFunc("/readyz", health.Readiness)
	adminSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: adminMux}

	errCh := make(chan error, 2)
	go func() {
		logger.Info("redirect server listening", "addr", cfg.HTTPAddr)
		if err := redirectSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		logger.Info("admin server listening", "addr", cfg.MetricsAddr)
		if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "err", err)
	}

	// Graceful shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = redirectSrv.Shutdown(shutdownCtx)
	_ = adminSrv.Shutdown(shutdownCtx)
	if err := emitter.Close(shutdownCtx); err != nil {
		logger.Warn("emitter close", "err", err)
	}
	if err := traceShutdown(shutdownCtx); err != nil {
		logger.Warn("trace shutdown", "err", err)
	}
	logger.Info("shutdown complete")
	return nil
}

// repoOrNil avoids handing a typed-nil *repo.Repo to the resolver interface.
func repoOrNil(r *repo.Repo) resolver.LinkRepo {
	if r == nil {
		return nil
	}
	return r
}
