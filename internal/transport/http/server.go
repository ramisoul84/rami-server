package http

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/ramisoul84/rami-server/internal/config"
	"github.com/ramisoul84/rami-server/internal/transport/http/handler"
	"github.com/ramisoul84/rami-server/internal/transport/http/middleware"
	"github.com/ramisoul84/rami-server/pkg/jwt"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

// bodyLimit caps request bodies; the largest legitimate one is an analytics event (~2 KB).
const bodyLimit = 64 * 1024

type Server struct {
	app *fiber.App
	cfg *config.Config
	log *logger.Logger
}

type Handlers struct {
	Auth  *handler.AuthHandler
	Event *handler.EventHandler
}

func NewServer(cfg *config.Config, log *logger.Logger, tokens *jwt.TokenManager, h Handlers) *Server {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.HTTP.ReadTimeout,
		WriteTimeout:          cfg.HTTP.WriteTimeout,
		IdleTimeout:           cfg.HTTP.IdleTimeout,
		BodyLimit:             bodyLimit,
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler(log),
	})

	app.Use(middleware.RequestID())
	app.Use(middleware.Logging(log))
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.HTTP.CORSOrigins,
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	api := app.Group("/api/v1")

	// Public: the tracker sends a few events per minute per visitor; login is brute-force bait.
	api.Post("/analytics/event", rateLimit(240, time.Minute), h.Event.Track)
	api.Post("/auth/login", rateLimit(5, time.Minute), h.Auth.Login)

	auth := middleware.JWTAuth(tokens)
	api.Post("/auth/logout", auth, h.Auth.Logout)

	return &Server{app: app, cfg: cfg, log: log}
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

// rateLimit allows max requests per window from one client IP.
func rateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          max,
		Expiration:   window,
		KeyGenerator: middleware.ClientIP,
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many requests, try again later")
		},
	})
}

// errorHandler renders every error as JSON. Internal causes are logged, never sent.
func errorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code, message := fiber.StatusInternalServerError, "internal server error"

		var fe *fiber.Error
		var ie *handler.InternalError
		switch {
		case errors.As(err, &fe):
			code, message = fe.Code, fe.Message
		case errors.As(err, &ie):
			message = ie.Message
		}

		requestID, _ := c.Locals("request_id").(string)
		if code >= 500 {
			log.Error("request failed",
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
				"message": message,
			},
			"request_id": requestID,
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
	case code == fiber.StatusRequestEntityTooLarge:
		return "payload_too_large"
	case code == fiber.StatusTooManyRequests:
		return "rate_limited"
	case code >= 500:
		return "internal"
	default:
		return "error"
	}
}
