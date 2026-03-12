package inbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

// CreateTestCaseUseCase: HandleWithParamsAndBody → fn(ctx, claims, P, B) (Res, err)
type CreateTestCaseUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest, body dto.CreateTestCaseRequest) (dto.CreateTestCaseResponse, error)
}

// ListTestCasesUseCase: HandleWithParamsAndClaims → fn(ctx, claims, Req) (Res, err)
type ListTestCasesUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.ProblemIDRequest) (dto.TestCaseListResponse, error)
}

// UpdateTestCaseUseCase: HandleVoidWithParamsAndBody → fn(ctx, claims, P, B) err
type UpdateTestCaseUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.TestCaseIDRequest, body dto.UpdateTestCaseRequest) error
}

// DeleteTestCaseUseCase: HandleVoidWithParamsAndClaims → fn(ctx, claims, Req) err
type DeleteTestCaseUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.TestCaseIDRequest) error
}
