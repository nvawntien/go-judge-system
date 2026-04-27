package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"go-judge-system/services/auth/internal/application/port/outbound"
)

type tokenGenerator struct{}

func NewTokenGenerator() outbound.TokenGenerator {
	return &tokenGenerator{}
}

// Generate creates a crypto-random token by combining the identifier with 32 random bytes.
func (g *tokenGenerator) Generate(identifier string) string {
	randomBytes := make([]byte, 32)
	_, _ = rand.Read(randomBytes)

	tokenData := append([]byte(identifier), randomBytes...)
	return hex.EncodeToString(tokenData)
}

// Hash produces a SHA-256 hash of the raw token for safe storage.
func (g *tokenGenerator) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
