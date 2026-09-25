package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramisoul84/rami-server/pkg/jwt"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrEmptyCredentials   = errors.New("auth: username and password are required")
)

type LoginResult struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

type AuthService interface {
	Login(ctx context.Context, username, password string) (*LoginResult, error)
}

type authService struct {
	tokens            *jwt.TokenManager
	adminUsername     string
	adminPasswordHash string
	log               *logger.Logger
}

func NewAuthService(
	tokens *jwt.TokenManager,
	adminUsername, adminPasswordHash string,
	log *logger.Logger,
) AuthService {
	return &authService{
		tokens:            tokens,
		adminUsername:     adminUsername,
		adminPasswordHash: adminPasswordHash,
		log:               log,
	}
}

func (s *authService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return nil, ErrEmptyCredentials
	}

	if !constantTimeEqual(username, s.adminUsername) {
		s.log.Warn("login failed: unknown username", "username", username)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(s.adminPasswordHash), []byte(password)); err != nil {
		s.log.Warn("login failed: wrong password", "username", username)
		return nil, ErrInvalidCredentials
	}

	token, err := s.tokens.Generate()
	if err != nil {
		s.log.Error("token generation failed", "err", err)
		return nil, fmt.Errorf("auth: generate token: %w", err)
	}

	s.log.Info("admin logged in", "username", username)

	return &LoginResult{
		Token:     token,
		ExpiresIn: int(s.tokens.AccessDuration().Seconds()),
	}, nil
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
