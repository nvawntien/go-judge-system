package problem

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"

	"go.uber.org/zap"
)

// public list problems
type listProblemsUseCase struct {
	problemRepo outbound.ProblemRepository
	logger      *zap.Logger
}

func NewListProblemsUseCase(problemRepo outbound.ProblemRepository, logger *zap.Logger) inbound.ListProblemsUseCase {
	return &listProblemsUseCase{problemRepo: problemRepo, logger: logger}
}

func (uc *listProblemsUseCase) Execute(ctx context.Context, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error) {
	offset := (req.Page - 1) * req.Limit

	problems, err := uc.problemRepo.List(ctx, offset, req.Limit, req.Difficulty, req.Search, false)
	if err != nil {
		uc.logger.Error("failed to list problems", zap.Error(err))
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.problemRepo.Count(ctx, req.Difficulty, req.Search, false)
	if err != nil {
		uc.logger.Error("failed to count problems", zap.Error(err))
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.ProblemResponse, 0, len(problems))
	for _, p := range problems {
		items = append(items, usecase.MapProblemToResponse(p, false))
	}

	return dto.ListProblemsResponse{Items: items, Total: total, Page: req.Page, Limit: req.Limit}, nil
}

// only super admin get this list problems
func (uc *listProblemsUseCase) ExecuteAdmin(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error) {
	if !claims.IsSuperAdmin() {
		return dto.ListProblemsResponse{}, domain.ErrForbidden
	}

	offset := (req.Page - 1) * req.Limit

	problems, err := uc.problemRepo.List(ctx, offset, req.Limit, req.Difficulty, req.Search, true)
	if err != nil {
		uc.logger.Error("failed to list problems (admin)", zap.Error(err))
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.problemRepo.Count(ctx, req.Difficulty, req.Search, true)
	if err != nil {
		uc.logger.Error("failed to count problems (admin)", zap.Error(err))
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.ProblemResponse, 0, len(problems))
	for _, p := range problems {
		items = append(items, usecase.MapProblemToResponse(p, true))
	}

	return dto.ListProblemsResponse{Items: items, Total: total, Page: req.Page, Limit: req.Limit}, nil
}

// only owner problem can get this list problems
func (uc *listProblemsUseCase) ExecuteMy(ctx context.Context, claims auth.Claims, req dto.ListProblemsRequest) (dto.ListProblemsResponse, error) {
	if !claims.IsAdmin() {
		return dto.ListProblemsResponse{}, domain.ErrForbidden
	}

	offset := (req.Page - 1) * req.Limit

	problems, err := uc.problemRepo.ListByAuthor(ctx, claims.UserID, offset, req.Limit, req.Difficulty, req.Search)
	if err != nil {
		uc.logger.Error("failed to list my problems", zap.Error(err))
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.problemRepo.CountByAuthor(ctx, claims.UserID, req.Difficulty, req.Search)
	if err != nil {
		uc.logger.Error("failed to count my problems", zap.Error(err))
		return dto.ListProblemsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.ProblemResponse, 0, len(problems))
	for _, p := range problems {
		items = append(items, usecase.MapProblemToResponse(p, true))
	}

	return dto.ListProblemsResponse{Items: items, Total: total, Page: req.Page, Limit: req.Limit}, nil
}
