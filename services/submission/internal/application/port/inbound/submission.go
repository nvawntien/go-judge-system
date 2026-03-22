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
	Execute(ctx context.Context, req dto.ListSubmissionsRequest) (dto.ListSubmissionsResponse, error)
	ExecuteMy(ctx context.Context, claims auth.Claims, req dto.ListMySubmissionsRequest) (dto.ListMySubmissionsResponse, error)
	ExecuteProblem(ctx context.Context, params dto.ProblemIDRequest, query dto.ListProblemSubmissionsQueryRequest) (dto.ListProblemSubmissionsResponse, error)
}

type GetSubmissionUseCase interface {
	ExecuteMy(ctx context.Context, claims auth.Claims, req dto.SubmissionIDRequest) (dto.SubmissionDetailResponse, error)
	ExecuteAdmin(ctx context.Context, claims auth.Claims, req dto.SubmissionIDRequest) (dto.SubmissionDetailResponse, error)
}

type RejudgeSubmissionUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.SubmissionIDRequest) error
}

type ProcessJudgeResultUseCase interface {
	Execute(ctx context.Context, message dto.JudgeResultMessage) error
}
