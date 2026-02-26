package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

type tokenGenerator struct {
}

func NewTokenGenerator() outbound.TokenGenerator {
	return &tokenGenerator{}
}

func (g *tokenGenerator) GenerateSecureToken(userID string) string {
	randomBytes := make([]byte, 32)
	_, _= rand.Read(randomBytes)

	tokenData := append([]byte(userID), randomBytes...)
	return hex.EncodeToString(tokenData)
}

func (g *tokenGenerator) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
