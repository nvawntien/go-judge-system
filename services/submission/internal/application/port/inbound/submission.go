package inbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
)

type CreateSubmissionUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateSubmissionRequest) (dto.SubmissionResponse, error)
}

type ListSubmissionsUseCase interface {
	ExecuteMy(ctx context.Context, claims auth.Claims, req dto.ListMySubmissionsRequest) (dto.ListMySubmissionsResponse, error)
	ExecuteProblem(ctx context.Context, params dto.ProblemIDRequest, query dto.ListProblemSubmissionsQueryRequest) (dto.ListProblemSubmissionsResponse, error)
}

type GetSubmissionUseCase interface {
	ExecuteMy(ctx context.Context, claims auth.Claims, req dto.SubmissionIDRequest) (dto.SubmissionDetailResponse, error)
}
