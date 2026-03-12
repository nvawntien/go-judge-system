package inbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

// CreateProblemUseCase: HandleWithClaims → fn(ctx, claims, Req) (Res, err)
type CreateProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateProblemRequest) (dto.CreateProblemResponse, error)
}

// UpdateProblemUseCase: HandleVoidWithParamsAndBody → fn(ctx, claims, P, B) err
type UpdateProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, body dto.UpdateProblemRequest) error
}

// DeleteProblemUseCase: HandleVoidWithParamsAndClaims → fn(ctx, claims, Req) err
type DeleteProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error
}

type GetProblemUseCase interface {
	// GetProblemUseCase (public by slug): HandleWithParams → fn(ctx, Req) (Res, err)
	Execute(ctx context.Context, params dto.ProblemSlugRequest) (dto.ProblemDetailResponse, error)
	// GetProblemAdminUseCase (admin by ID): HandleWithParamsAndClaims → fn(ctx, claims, Req) (Res, err)
	ExecuteAdmin(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.ProblemDetailResponse, error)
}

type ListProblemsUseCase interface {
	// ListProblemsUseCase (public): HandleWithQuery → fn(ctx, Req) (Res, err)
	Execute(ctx context.Context, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error)
	// ListProblemsAdminUseCase (admin all): HandleWithQueryAndClaims → fn(ctx, claims, Req) (Res, err)
	ExecuteAdmin(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error)
	// ListMyProblemsUseCase (my problems): HandleWithQueryAndClaims → fn(ctx, claims, Req) (Res, err)
	ExecuteMy(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error)
}

// PublishProblemUseCase: HandleVoidWithParamsAndClaims → fn(ctx, claims, Req) err
type PublishProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error
}

// HideProblemUseCase: HandleVoidWithParamsAndClaims → fn(ctx, claims, Req) err
type HideProblemUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) error
}
