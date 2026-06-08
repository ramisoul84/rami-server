package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ramisoul84/rami-server/internal/config"
	"github.com/ramisoul84/rami-server/internal/notifier"
	"github.com/ramisoul84/rami-server/internal/repository"
	"github.com/ramisoul84/rami-server/internal/service"
	transporthttp "github.com/ramisoul84/rami-server/internal/transport/http"
	"github.com/ramisoul84/rami-server/internal/transport/http/handler"
	"github.com/ramisoul84/rami-server/migrations"
	"github.com/ramisoul84/rami-server/pkg/database"
	"github.com/ramisoul84/rami-server/pkg/jwt"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

type App struct {
	log    *logger.Logger
	db     *database.Postgres
	geo    service.GeoService
	server *transporthttp.Server
	bot    notifier.Runnable // nil when Telegram is not configured
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
	log.Info("starting application", "name", cfg.App.Name, "version", cfg.App.Version, "env", cfg.App.Environment)

	db, err := database.NewPostgres(&cfg.DB, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	log.Info("postgres connected")

	migrateCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	applied, err := migrations.Up(migrateCtx, db.DB)
	cancel()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if len(applied) > 0 {
		log.Info("migrations applied", "versions", applied)
	}

	tokens, err := jwt.New(cfg.Auth.JWTSecret, cfg.Auth.AccessDuration, cfg.App.Name)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("jwt: %w", err)
	}

	// Geo is optional: without the MMDB file visits are stored without a country.
	var geo service.GeoService
	if cfg.Geo.MMDBPath != "" {
		if g, err := service.NewGeoService(cfg.Geo.MMDBPath, log); err != nil {
			log.Warn("geo service disabled", "err", err)
		} else {
			geo = g
			log.Info("geo service enabled", "mmdb", cfg.Geo.MMDBPath)
		}
	}

	// Times in bot messages and the digest schedule follow the DB timezone (validated in config).
	loc, _ := time.LoadLocation(cfg.DB.Timezone)

	eventRepo := repository.NewEventRepository(db.DB)
	analyticsService := service.NewAnalyticsService(repository.NewAnalyticsRepository(db.DB), log)
	authService := service.NewAuthService(tokens, cfg.Auth.AdminUsername, cfg.Auth.AdminPasswordHash, log)

	bot, err := notifier.NewTelegramNotifier(
		notifier.TelegramConfig{
			BotToken:   cfg.Telegram.BotToken,
			Admins:     cfg.Telegram.Admins,
			SiteURL:    cfg.Telegram.SiteURL,
			DigestHour: cfg.Telegram.DigestHour,
			Location:   loc,
		},
		log,
		analyticsService,
		repository.NewTelegramRepository(db.DB),
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("notifier: %w", err)
	}

	a := &App{log: log, db: db, geo: geo}
	var visitNotifier service.VisitNotifier
	if bot.Enabled() {
		visitNotifier, a.bot = bot, bot
		log.Info("telegram bot enabled", "admins", len(cfg.Telegram.Admins))
	} else {
		log.Warn("telegram bot disabled: TELEGRAM_BOT_TOKEN is empty")
	}

	a.server = transporthttp.NewServer(cfg, log, tokens, transporthttp.Handlers{
		Auth:      handler.NewAuthHandler(authService),
		Event:     handler.NewEventHandler(service.NewEventService(eventRepo, visitNotifier, geo, log)),
		Analytics: handler.NewAnalyticsHandler(analyticsService),
	})
	return a, nil
}

func (a *App) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := a.server.Start(); err != nil {
			errCh <- err
		}
	}()

	if a.bot != nil {
		go func() {
			if err := a.bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				a.log.Error("telegram bot stopped", "err", err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	a.log.Info("shutting down")
	var errs []error
	if err := a.server.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("http shutdown: %w", err))
	}
	if a.geo != nil {
		_ = a.geo.Close()
	}
	if err := a.db.Close(); err != nil {
		errs = append(errs, fmt.Errorf("db close: %w", err))
	}
	a.log.Info("shutdown complete")
	return errors.Join(errs...)
}
