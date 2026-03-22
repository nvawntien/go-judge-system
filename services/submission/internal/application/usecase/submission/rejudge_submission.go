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
	txManager            outbound.TransactionManager
	submissionRepo       outbound.SubmissionRepository
	submissionResultRepo outbound.SubmissionResultRepository
	problemAccessChecker outbound.ProblemAccessChecker
	judgePublisher       outbound.JudgePublisher
	logger               *zap.Logger
}

func NewRejudgeSubmissionUseCase(
	txManager outbound.TransactionManager,
	submissionRepo outbound.SubmissionRepository,
	submissionResultRepo outbound.SubmissionResultRepository,
	problemAccessChecker outbound.ProblemAccessChecker,
	judgePublisher outbound.JudgePublisher,
	logger *zap.Logger,
) inbound.RejudgeSubmissionUseCase {
	return &rejudgeSubmissionUseCase{
		txManager:            txManager,
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

	err = uc.txManager.ExecuteInTx(ctx, func(txCtx context.Context) error {
		if err := uc.submissionRepo.Update(txCtx, submission); err != nil {
			return err
		}

		if err := uc.submissionResultRepo.DeleteBySubmissionID(txCtx, submission.ID); err != nil {
			return err
		}

		if err := uc.judgePublisher.Publish(txCtx, submission); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		uc.logger.Error("failed to rejudge submission or write to outbox", zap.Int64("submission_id", req.ID), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}
