package jwt

import (
	"context"
	"fmt"
	"go-judge-system/pkg/config"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwtProvider struct {
	accessSecret    string
	refreshSecret   string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTProvider(cfg config.JWTConfig) outbound.JWTProvider {
	return &jwtProvider{
		accessSecret:    cfg.AccessSecret,
		refreshSecret:   cfg.RefreshSecret,
		accessTokenTTL:  cfg.AccessTTL,
		refreshTokenTTL: cfg.RefreshTTL,
	}
}

type customClaims struct {
	Role     string `json:"role"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (p *jwtProvider) GenerateAccessToken(ctx context.Context, userID, username, role string) (string, int, error) {
	claims := customClaims{
		Role:     role,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth-service",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(p.accessSecret))
	if err != nil {
		return "", 0, err
	}

	return token, int(p.accessTokenTTL.Seconds()), nil
}

func (p *jwtProvider) GenerateRefreshToken(ctx context.Context, userID, username, role string) (string, int, error) {
	claims := customClaims{
		Role:     role,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth-service",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(p.refreshSecret))
	if err != nil {
		return "", 0, err
	}

	return token, int(p.refreshTokenTTL.Seconds()), nil
}

func (p *jwtProvider) VerifyAccessToken(ctx context.Context, tokenStr string) (string, string, string, error) {
	return p.verifyToken(ctx, tokenStr, p.accessSecret)
}

func (p *jwtProvider) VerifyRefreshToken(ctx context.Context, tokenStr string) (string, string, string, error) {
	return p.verifyToken(ctx, tokenStr, p.refreshSecret)
}

func (p *jwtProvider) verifyToken(ctx context.Context, tokenStr string, secret string) (string, string, string, error) {
	claims := &customClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return "", "", "", fmt.Errorf("token invalid or expired")
	}
	return claims.Subject, claims.Username, claims.Role, nil
}
