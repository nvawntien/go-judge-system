package submission

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/application/usecase"
	"go-judge-system/services/submission/internal/domain"

	"go.uber.org/zap"
)

type getSubmissionUseCase struct {
	submissionRepo       outbound.SubmissionRepository
	submissionResultRepo outbound.SubmissionResultRepository
	logger               *zap.Logger
}

func NewGetSubmissionUseCase(
	submissionRepo outbound.SubmissionRepository,
	submissionResultRepo outbound.SubmissionResultRepository,
	logger *zap.Logger,
) inbound.GetSubmissionUseCase {
	return &getSubmissionUseCase{
		submissionRepo:       submissionRepo,
		submissionResultRepo: submissionResultRepo,
		logger:               logger,
	}
}

func (uc *getSubmissionUseCase) ExecuteMy(ctx context.Context, claims auth.Claims, req dto.SubmissionIDRequest) (dto.SubmissionDetailResponse, error) {
	submission, err := uc.submissionRepo.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, domain.ErrSubmissionNotFound) {
			return dto.SubmissionDetailResponse{}, domain.ErrSubmissionNotFound
		}

		uc.logger.Error("failed to get submission", zap.Int64("submission_id", req.ID), zap.Error(err))
		return dto.SubmissionDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if submission.UserID != claims.UserID {
		return dto.SubmissionDetailResponse{}, domain.ErrForbidden
	}

	results, err := uc.submissionResultRepo.GetBySubmissionID(ctx, submission.ID)
	if err != nil {
		uc.logger.Error("failed to get submission results", zap.Int64("submission_id", submission.ID), zap.Error(err))
		return dto.SubmissionDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return usecase.MapSubmissionToDetailResponse(submission, results), nil
}
