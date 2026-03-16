package inbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
)

type CreateSubmissionUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateSubmissionRequest) (dto.CreateSubmissionResponse, error)
}
