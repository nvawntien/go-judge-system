package submission

import (
	"context"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/application/usecase"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"go.uber.org/zap"
)

type listSubmissionsUseCase struct {
	submissionRepo outbound.SubmissionRepository
	logger         *zap.Logger
}

func NewListSubmissionsUseCase(submissionRepo outbound.SubmissionRepository, logger *zap.Logger) inbound.ListSubmissionsUseCase {
	return &listSubmissionsUseCase{submissionRepo: submissionRepo, logger: logger}
}

func (uc *listSubmissionsUseCase) ExecuteMy(ctx context.Context, claims auth.Claims, req dto.ListMySubmissionsRequest) (dto.ListMySubmissionsResponse, error) {
	offset := (req.Page - 1) * req.Limit
	status := strings.ToUpper(req.Status)
	language := strings.ToUpper(req.Language)

	if status != "" && !isValidStatus(status) {
		return dto.ListMySubmissionsResponse{}, domain.ErrInvalidStatus
	}

	if language != "" {
		if _, ok := entity.ParseLanguage(language); !ok {
			return dto.ListMySubmissionsResponse{}, domain.ErrInvalidLanguage
		}
	}

	submissions, err := uc.submissionRepo.ListByUser(ctx, claims.UserID, offset, req.Limit, status, language)
	if err != nil {
		uc.logger.Error("failed to list my submissions", zap.Error(err))
		return dto.ListMySubmissionsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.submissionRepo.CountByUser(ctx, claims.UserID, status, language)
	if err != nil {
		uc.logger.Error("failed to count my submissions", zap.Error(err))
		return dto.ListMySubmissionsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.SubmissionResponse, 0, len(submissions))
	for _, s := range submissions {
		items = append(items, usecase.MapSubmissionToResponse(s))
	}

	return dto.ListMySubmissionsResponse{Items: items, Total: total, Page: req.Page, Limit: req.Limit}, nil
}

func (uc *listSubmissionsUseCase) ExecuteProblem(ctx context.Context, params dto.ProblemIDRequest, query dto.ListProblemSubmissionsQueryRequest) (dto.ListProblemSubmissionsResponse, error) {
	offset := (query.Page - 1) * query.Limit
	status := strings.ToUpper(query.Status)
	language := strings.ToUpper(query.Language)

	if status != "" && !isValidStatus(status) {
		return dto.ListProblemSubmissionsResponse{}, domain.ErrInvalidStatus
	}

	if language != "" {
		if _, ok := entity.ParseLanguage(language); !ok {
			return dto.ListProblemSubmissionsResponse{}, domain.ErrInvalidLanguage
		}
	}

	submissions, err := uc.submissionRepo.ListByProblem(ctx, params.ID, offset, query.Limit, status, language)
	if err != nil {
		uc.logger.Error("failed to list submissions by problem", zap.Int64("problem_id", params.ID), zap.Error(err))
		return dto.ListProblemSubmissionsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	total, err := uc.submissionRepo.CountByProblem(ctx, params.ID, status, language)
	if err != nil {
		uc.logger.Error("failed to count submissions by problem", zap.Int64("problem_id", params.ID), zap.Error(err))
		return dto.ListProblemSubmissionsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.SubmissionResponse, 0, len(submissions))
	for _, s := range submissions {
		items = append(items, usecase.MapSubmissionToResponse(s))
	}

	return dto.ListProblemSubmissionsResponse{Items: items, Total: total, Page: query.Page, Limit: query.Limit}, nil
}

func isValidStatus(status string) bool {
	switch entity.Status(status) {
	case entity.StatusPending,
		entity.StatusJudging,
		entity.StatusAccepted,
		entity.StatusWrongAnswer,
		entity.StatusTimeLimitExceed,
		entity.StatusMemoryLimitExceed,
		entity.StatusRuntimeError,
		entity.StatusCompilationError:
		return true
	default:
		return false
	}
}
