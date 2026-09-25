package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/rami-server/internal/service"
)

type AuthHandler struct {
	auth service.AuthService
}

func NewAuthHandler(auth service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	result, err := h.auth.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyCredentials):
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "login failed")
		}
	}
	return c.JSON(result)
}
