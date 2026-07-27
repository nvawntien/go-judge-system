package inbound

import (
	"context"

	"go-judge-system/pkg/rbac"
)

type GetProblemRequest struct {
	ProblemID   int64
	ActorUserID string
	ActorRole   rbac.Role
}

type ProblemMetadata struct {
	ID          int64
	Title       string
	Slug        string
	TimeLimit   float64
	MemoryLimit int
}

type GetProblemUseCase interface {
	Execute(ctx context.Context, req GetProblemRequest) (ProblemMetadata, error)
}
