package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/rami-server/pkg/jwt"
)

func JWTAuth(tm *jwt.TokenManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}
		token := strings.TrimPrefix(header, "Bearer ")
		if err := tm.Validate(token); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return c.Next()
	}
}
