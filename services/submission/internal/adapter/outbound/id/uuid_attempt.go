package id

import (
	"go-judge-system/services/submission/internal/application/port/outbound"

	"github.com/google/uuid"
)

type uuidAttemptIDGenerator struct{}

func NewUUIDAttemptIDGenerator() outbound.AttemptIDGenerator {
	return uuidAttemptIDGenerator{}
}

func (uuidAttemptIDGenerator) NewAttemptID() string {
	return uuid.New().String()
}
