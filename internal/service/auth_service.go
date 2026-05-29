package service

import (
	"context"
	"crypto/subtle"
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
	if username == "" || password == "" {
		return nil, ErrEmptyCredentials
	}

	// Always run bcrypt, even for a wrong username, so response time doesn't
	// reveal which of the two was wrong.
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.adminUsername)) == 1
	hash := []byte(s.adminPasswordHash)
	if !userOK {
		hash = dummyHash
	}
	passOK := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil

	if !userOK || !passOK {
		s.log.Warn("login failed", "username", username)
		return nil, ErrInvalidCredentials
	}

	token, err := s.tokens.Generate()
	if err != nil {
		return nil, fmt.Errorf("auth: generate token: %w", err)
	}

	s.log.Info("admin logged in", "username", username)
	return &LoginResult{
		Token:     token,
		ExpiresIn: int(s.tokens.AccessDuration().Seconds()),
	}, nil
}

// dummyHash is compared against when the username is wrong (bcrypt of a random string).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-password"), bcrypt.DefaultCost)
