package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

type resetTokenGenerator struct {
}

func NewResetTokenGenerator() outbound.ResetTokenGenerator {
	return &resetTokenGenerator{}
}

func (g *resetTokenGenerator) Generate(userID string) string{
	randomBytes := make([]byte, 32)
	_, _= rand.Read(randomBytes)

	tokenData := append([]byte(userID), randomBytes...)
	return hex.EncodeToString(tokenData)
}

func (g *resetTokenGenerator) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
