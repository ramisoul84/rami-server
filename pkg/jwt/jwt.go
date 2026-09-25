package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("jwt: invalid token")
	ErrExpiredToken = errors.New("jwt: token expired")
)

type TokenManager struct {
	secretKey      []byte
	accessDuration time.Duration
	issuer         string
}

func New(secret string, accessDuration time.Duration, issuer string) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt: secret must be >= 32 chars, got %d", len(secret))
	}
	if accessDuration <= 0 {
		return nil, errors.New("jwt: access duration must be positive")
	}
	return &TokenManager{
		secretKey:      []byte(secret),
		accessDuration: accessDuration,
		issuer:         issuer,
	}, nil
}

func (m *TokenManager) Generate() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   "admin",
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.accessDuration)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

func (m *TokenManager) Validate(tokenString string) error {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) { return m.secretKey, nil },
		jwt.WithIssuer(m.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrExpiredToken
		}
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return ErrInvalidToken
	}
	return nil
}

func (m *TokenManager) AccessDuration() time.Duration { return m.accessDuration }
