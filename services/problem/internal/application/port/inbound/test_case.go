package inbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

type UploadTestCaseUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, req dto.UploadTestCaseRequest) (dto.UploadTestCasesResponse, error)
}