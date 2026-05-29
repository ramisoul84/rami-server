package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ramisoul84/rami-server/internal/config"
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

	eventRepo := repository.NewEventRepository(db.DB)
	authService := service.NewAuthService(tokens, cfg.Auth.AdminUsername, cfg.Auth.AdminPasswordHash, log)

	a := &App{log: log, db: db, geo: geo}
	a.server = transporthttp.NewServer(cfg, log, tokens, transporthttp.Handlers{
		Auth:  handler.NewAuthHandler(authService),
		Event: handler.NewEventHandler(service.NewEventService(eventRepo, nil, geo, log)),
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
