package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/ramisoul84/rami-server/internal/config"
	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/notifier"
	"github.com/ramisoul84/rami-server/internal/repository"
	"github.com/ramisoul84/rami-server/internal/service"
	transporthttp "github.com/ramisoul84/rami-server/internal/transport/http"
	"github.com/ramisoul84/rami-server/internal/transport/http/handler"
	"github.com/ramisoul84/rami-server/pkg/database"
	"github.com/ramisoul84/rami-server/pkg/jwt"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

type App struct {
	config   *config.Config
	log      *logger.Logger
	db       *database.Postgres
	geo      service.GeoService
	server   *transporthttp.Server
	runnable notifier.Runnable
}

func New(cfg *config.Config) (*App, error) {
	log := logger.New(&logger.Config{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		Output:     cfg.Logger.Output,
		FilePath:   cfg.Logger.FilePath,
		Service:    cfg.Logger.Service,
		MaxSizeMB:  cfg.Logger.MaxSizeMB,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAgeDays: cfg.Logger.MaxAgeDays,
		Compress:   cfg.Logger.Compress,
	})

	log.Info("starting application",
		"name", cfg.App.Name,
		"version", cfg.App.Version,
		"env", cfg.App.Environment,
	)

	// Database
	db, err := database.NewPostgres(&cfg.DB, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	log.Info("postgres connected")

	// JWT
	tokenManager, err := jwt.New(cfg.Auth.JWTSecret, cfg.Auth.AccessDuration, cfg.App.Name)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("jwt: %w", err)
	}

	// Geo (optional — silently disabled if MMDB is missing)
	var geo service.GeoService
	if cfg.Geo.MMDBPath != "" {
		g, err := service.NewGeoService(cfg.Geo.MMDBPath, log)
		if err != nil {
			log.Warn("geo service disabled", "err", err)
		} else {
			geo = g
			log.Info("geo service enabled", "mmdb", cfg.Geo.MMDBPath)
		}
	}

	// Repositories
	eventRepo := repository.NewEventRepository(db.DB)
	telegramRepo := repository.NewTelegramRepository(db.DB)

	// Services
	authService := service.NewAuthService(
		tokenManager,
		cfg.Auth.AdminUsername,
		cfg.Auth.AdminPasswordHash,
		log,
	)
	statsService := service.NewStatsService(eventRepo, log)

	statsProvider := func(ctx context.Context) (*domain.DashboardStats, error) {
		return statsService.Overview(ctx, domain.DateRange{})
	}
	visitsProvider := func(ctx context.Context, limit int) ([]domain.Visit, error) {
		return statsService.RecentVisits(ctx, limit)
	}

	// Notifier
	n, err := notifier.NewTelegramNotifier(
		cfg.Telegram.BotToken,
		cfg.Telegram.Admins,
		log,
		statsProvider,
		visitsProvider,
		telegramRepo,
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("notifier: %w", err)
	}

	var visitNotifier service.VisitNotifier
	if n != nil && n.Enabled() {
		visitNotifier = n
		log.Info("telegram notifier enabled", "admins", len(cfg.Telegram.Admins))
	} else {
		log.Warn("telegram notifier disabled")
	}

	eventService := service.NewEventService(eventRepo, visitNotifier, geo, log)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	eventHandler := handler.NewEventHandler(eventService)
	statsHandler := handler.NewStatsHandler(statsService)

	// HTTP server
	server := transporthttp.NewServer(
		cfg, log, tokenManager,
		authHandler, eventHandler, statsHandler,
	)

	var runnable notifier.Runnable
	if n != nil {
		runnable = n
	}

	return &App{
		config:   cfg,
		log:      log,
		db:       db,
		geo:      geo,
		server:   server,
		runnable: runnable,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.log.Info("starting transports")

	errCh := make(chan error, 1)

	go func() {
		if err := a.server.Start(); err != nil {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	if a.runnable != nil {
		go func() {
			if err := a.runnable.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				a.log.Error("notifier run stopped", "err", err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
		return nil
	case err := <-errCh:
		a.log.Error("transport failed", "err", err)
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	a.log.Info("shutting down")
	var errs []error

	if err := a.server.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("http shutdown: %w", err))
	}
	if closer, ok := a.geo.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("db close: %w", err))
		}
	}
	a.log.Info("shutdown complete")
	return errors.Join(errs...)
}
