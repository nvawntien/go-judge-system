package user

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
)

type CreateSubmissionUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateSubmissionRequest) (dto.CreateSubmissionResponse, error)
}

type RunCodeUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.RunCodeRequest) (dto.RunCodeResponse, error)
}

type GetSubmissionUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.GetSubmissionRequest) (dto.GetSubmissionResponse, error)
}

type ListMySubmissionsUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.ListMySubmissionsRequest) (dto.ListMySubmissionsResponse, error)
}

type GetMyProfileStatsUseCase interface {
	Execute(ctx context.Context, claims auth.Claims) (dto.GetMyProfileStatsResponse, error)
}

type IssueSubmissionStreamTicketUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.IssueSubmissionStreamTicketRequest) (dto.IssueSubmissionStreamTicketResponse, error)
}
