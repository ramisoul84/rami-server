package http

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/ramisoul84/rami-server/internal/config"
	"github.com/ramisoul84/rami-server/internal/transport/http/handler"
	"github.com/ramisoul84/rami-server/internal/transport/http/middleware"
	"github.com/ramisoul84/rami-server/pkg/jwt"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

type Server struct {
	app *fiber.App
	cfg *config.Config
	log *logger.Logger

	tokenManager *jwt.TokenManager

	authHandler  *handler.AuthHandler
	eventHandler *handler.EventHandler
	statsHandler *handler.StatsHandler
}

func NewServer(
	cfg *config.Config,
	log *logger.Logger,
	tokenManager *jwt.TokenManager,
	authHandler *handler.AuthHandler,
	eventHandler *handler.EventHandler,
	statsHandler *handler.StatsHandler,
) *Server {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.HTTP.ReadTimeout,
		WriteTimeout:          cfg.HTTP.WriteTimeout,
		IdleTimeout:           cfg.HTTP.IdleTimeout,
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler(log),
	})

	s := &Server{
		app:          app,
		cfg:          cfg,
		log:          log,
		tokenManager: tokenManager,
		authHandler:  authHandler,
		eventHandler: eventHandler,
		statsHandler: statsHandler,
	}
	s.registerMiddleware()
	s.registerRoutes()
	return s
}

func (s *Server) Start() error {
	addr := ":" + s.cfg.HTTP.Port
	s.log.Info("http server listening", "addr", addr)
	if err := s.app.Listen(addr); err != nil {
		return fmt.Errorf("http server failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

func (s *Server) registerMiddleware() {
	s.app.Use(middleware.RequestID())
	s.app.Use(middleware.Logging(s.log))
	s.app.Use(recover.New())
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     s.cfg.HTTP.CORSOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		AllowCredentials: true,
	}))
}

func (s *Server) registerRoutes() {
	s.app.Get("/health", s.healthCheck)

	api := s.app.Group("/api/v1")

	api.Post("/auth/login", s.authHandler.Login)
	api.Post("/analytics/event", s.eventHandler.Track)

	admin := api.Group("/admin", middleware.JWTAuth(s.tokenManager))
	admin.Get("/stats", s.statsHandler.Stats)
	admin.Get("/stats/live", s.statsHandler.LiveVisitors)
	admin.Get("/stats/countries", s.statsHandler.TopCountries)
	admin.Get("/visits/recent", s.statsHandler.RecentVisits)
	admin.Get("/export.csv", s.statsHandler.ExportCSV)
}

func (s *Server) healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func errorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}
		requestID, _ := c.Locals("request_id").(string)

		if code >= 500 {
			log.Error("unhandled error",
				"method", c.Method(),
				"path", c.Path(),
				"status", code,
				"err", err.Error(),
				"request_id", requestID,
			)
		}
		return c.Status(code).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"type":    errorType(code),
				"message": err.Error(),
			},
			"request_id": requestID,
			"timestamp":  time.Now().Unix(),
		})
	}
}

func errorType(code int) string {
	switch {
	case code == fiber.StatusUnauthorized:
		return "unauthorized"
	case code == fiber.StatusForbidden:
		return "forbidden"
	case code == fiber.StatusNotFound:
		return "not_found"
	case code == fiber.StatusBadRequest:
		return "bad_request"
	case code == fiber.StatusTooManyRequests:
		return "rate_limited"
	case code >= 500:
		return "internal"
	default:
		return "error"
	}
}
