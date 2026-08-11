package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-judge-system/pkg/auth"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/admin"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type rejudgeAdminSubmissionUseCase struct {
	submissionRepo outbound.SubmissionRepository
	attemptRepo    outbound.SubmissionAttemptRepository
	txManager      outbound.TransactionManager
	judgePublisher outbound.JudgePublisher
	attemptIDs     outbound.AttemptIDGenerator
	problemReader  outbound.ProblemReader
}

func NewRejudgeAdminSubmissionUseCase(
	submissionRepo outbound.SubmissionRepository,
	attemptRepo outbound.SubmissionAttemptRepository,
	txManager outbound.TransactionManager,
	judgePublisher outbound.JudgePublisher,
	attemptIDs outbound.AttemptIDGenerator,
	problemReader outbound.ProblemReader,
) inbound.RejudgeAdminSubmissionUseCase {
	return &rejudgeAdminSubmissionUseCase{
		submissionRepo: submissionRepo,
		attemptRepo:    attemptRepo,
		txManager:      txManager,
		judgePublisher: judgePublisher,
		attemptIDs:     attemptIDs,
		problemReader:  problemReader,
	}
}

func (uc *rejudgeAdminSubmissionUseCase) Execute(
	ctx context.Context,
	claims auth.Claims,
	req dto.RejudgeAdminSubmissionRequest,
) (dto.RejudgeAdminSubmissionResponse, error) {
	if strings.TrimSpace(claims.UserID) == "" || claims.Role == "" {
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrSubmissionUnauthenticated
	}
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrSubmissionForbidden
	}
	if req.SubmissionID <= 0 {
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrInvalidSubmissionID
	}

	submission, err := uc.submissionRepo.GetByID(ctx, req.SubmissionID)
	if err != nil {
		return dto.RejudgeAdminSubmissionResponse{}, mapRejudgeReadError(err)
	}
	if submission == nil {
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrSubmissionNotFound
	}
	if !entity.IsTerminalStatus(submission.Status) {
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrSubmissionRejudgeConflict
	}

	if _, err := uc.problemReader.GetTestCaseMetadata(ctx, submission.ProblemID); err != nil {
		return dto.RejudgeAdminSubmissionResponse{}, err
	}

	attemptID := strings.TrimSpace(uc.attemptIDs.NewAttemptID())
	if attemptID == "" {
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrInternalServer.Wrap(domain.ErrInvalidJudgeResult)
	}

	enqueuedAt := time.Now().UTC()
	err = uc.txManager.ExecuteInTx(ctx, func(txCtx context.Context) error {
		lockedSubmission, err := uc.submissionRepo.GetByIDForUpdate(txCtx, req.SubmissionID)
		if err != nil {
			return err
		}
		if !entity.IsTerminalStatus(lockedSubmission.Status) {
			return domain.ErrSubmissionRejudgeConflict
		}

		lockedSubmission.CurrentAttemptID = attemptID
		lockedSubmission.ResetForRejudge()
		if err := uc.submissionRepo.Update(txCtx, lockedSubmission); err != nil {
			return err
		}
		if err := uc.attemptRepo.Create(txCtx, entity.NewSubmissionAttempt(
			lockedSubmission.ID,
			attemptID,
			entity.AttemptTriggerAdminRejudge,
			claims.UserID,
		)); err != nil {
			return err
		}

		return uc.judgePublisher.Publish(txCtx, pkgjudge.JobMessage{
			SubmissionID: lockedSubmission.ID,
			ProblemID:    lockedSubmission.ProblemID,
			UserID:       lockedSubmission.UserID,
			Language:     string(lockedSubmission.Language),
			SourceCode:   lockedSubmission.SourceCode,
			AttemptID:    attemptID,
			EnqueuedAt:   enqueuedAt,
		})
	})
	if err != nil {
		if errors.Is(err, domain.ErrSubmissionNotFound) ||
			errors.Is(err, domain.ErrSubmissionRejudgeConflict) {
			return dto.RejudgeAdminSubmissionResponse{}, err
		}
		return dto.RejudgeAdminSubmissionResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.RejudgeAdminSubmissionResponse{
		SubmissionID:             req.SubmissionID,
		AttemptID:                attemptID,
		Status:                   string(entity.StatusPending),
		AttemptTrigger:           string(entity.AttemptTriggerAdminRejudge),
		AttemptTriggeredByUserID: claims.UserID,
		EnqueuedAt:               enqueuedAt,
	}, nil
}

func mapRejudgeReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, domain.ErrSubmissionNotFound) {
		return err
	}
	return domain.ErrInternalServer.Wrap(err)
}
