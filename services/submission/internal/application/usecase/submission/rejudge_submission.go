package submission

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"

	"go.uber.org/zap"
)

type rejudgeSubmissionUseCase struct {
	submissionRepo       outbound.SubmissionRepository
	submissionResultRepo outbound.SubmissionResultRepository
	problemAccessChecker outbound.ProblemAccessChecker
	judgePublisher       outbound.JudgePublisher
	logger               *zap.Logger
}

func NewRejudgeSubmissionUseCase(
	submissionRepo outbound.SubmissionRepository,
	submissionResultRepo outbound.SubmissionResultRepository,
	problemAccessChecker outbound.ProblemAccessChecker,
	judgePublisher outbound.JudgePublisher,
	logger *zap.Logger,
) inbound.RejudgeSubmissionUseCase {
	return &rejudgeSubmissionUseCase{
		submissionRepo:       submissionRepo,
		submissionResultRepo: submissionResultRepo,
		problemAccessChecker: problemAccessChecker,
		judgePublisher:       judgePublisher,
		logger:               logger,
	}
}

func (uc *rejudgeSubmissionUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.SubmissionIDRequest) error {
	if !claims.IsAdmin() {
		return domain.ErrForbidden
	}

	submission, err := uc.submissionRepo.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, domain.ErrSubmissionNotFound) {
			return domain.ErrSubmissionNotFound
		}

		uc.logger.Error("failed to get submission for rejudge", zap.Int64("submission_id", req.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	allowed, err := uc.problemAccessChecker.CanManageProblem(ctx, claims, submission.ProblemID)
	if err != nil {
		uc.logger.Error(
			"failed to verify problem management permission for rejudge",
			zap.Int64("submission_id", req.ID),
			zap.Int64("problem_id", submission.ProblemID),
			zap.String("user_id", claims.UserID),
			zap.Error(err),
		)
		return domain.ErrInternalServer.Wrap(err)
	}
	if !allowed {
		return domain.ErrForbidden
	}

	submission.ResetForRejudge()
	if err := uc.submissionRepo.Update(ctx, submission); err != nil {
		uc.logger.Error("failed to reset submission for rejudge", zap.Int64("submission_id", req.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.submissionResultRepo.DeleteBySubmissionID(ctx, submission.ID); err != nil {
		uc.logger.Error("failed to clear submission results for rejudge", zap.Int64("submission_id", req.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := uc.judgePublisher.Publish(ctx, submission); err != nil {
		uc.logger.Error("failed to republish submission to judge", zap.Int64("submission_id", req.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
