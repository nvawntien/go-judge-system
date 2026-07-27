package result

import (
	"context"
	"strings"

	pkgjudge "go-judge-system/pkg/judge"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/result"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type applyJudgeResultUseCase struct {
	submissionRepo outbound.SubmissionRepository
	resultRepo     outbound.SubmissionResultRepository
	txManager      outbound.TransactionManager
}

func NewApplyJudgeResultUseCase(
	submissionRepo outbound.SubmissionRepository,
	resultRepo outbound.SubmissionResultRepository,
	txManager outbound.TransactionManager,
) inbound.ApplyJudgeResultUseCase {
	return &applyJudgeResultUseCase{
		submissionRepo: submissionRepo,
		resultRepo:     resultRepo,
		txManager:      txManager,
	}
}

func (uc *applyJudgeResultUseCase) Execute(ctx context.Context, msg pkgjudge.ResultMessage) error {
	if msg.SubmissionID <= 0 || strings.TrimSpace(msg.AttemptID) == "" {
		return domain.ErrInvalidJudgeResult
	}

	status, err := mapTerminalSubmissionStatus(msg.Status)
	if err != nil {
		return err
	}

	results, err := mapTestCaseResults(msg.SubmissionID, msg.AttemptID, msg.TestCases)
	if err != nil {
		return err
	}

	return uc.txManager.ExecuteInTx(ctx, func(txCtx context.Context) error {
		submission, err := uc.submissionRepo.GetByIDForUpdate(txCtx, msg.SubmissionID)
		if err != nil {
			return err
		}

		if strings.TrimSpace(submission.CurrentAttemptID) == "" {
			return nil
		}
		if submission.CurrentAttemptID != msg.AttemptID {
			return nil
		}

		submission.MarkCompleted(status, msg.ExecutionTime, msg.MemoryUsed, msg.CompileOutput)
		if err := uc.submissionRepo.Update(txCtx, submission); err != nil {
			return err
		}

		return uc.resultRepo.ReplaceBySubmissionIDAndAttemptID(txCtx, msg.SubmissionID, msg.AttemptID, results)
	})
}

func mapTerminalSubmissionStatus(raw string) (entity.Status, error) {
	status, ok := entity.ParseStatus(raw)
	if !ok {
		return "", domain.ErrInvalidSubmissionStatus
	}

	switch status {
	case entity.StatusAccepted,
		entity.StatusWrongAnswer,
		entity.StatusTimeLimitExceed,
		entity.StatusMemoryLimitExceed,
		entity.StatusRuntimeError,
		entity.StatusCompilationError,
		entity.StatusSystemError:
		return status, nil
	default:
		return "", domain.ErrInvalidSubmissionStatus
	}
}

func mapTestCaseResults(submissionID int64, attemptID string, items []pkgjudge.TestCaseResultItem) ([]*entity.SubmissionResult, error) {
	results := make([]*entity.SubmissionResult, 0, len(items))
	for _, item := range items {
		if item.Index <= 0 {
			return nil, domain.ErrInvalidJudgeResult
		}

		status, err := mapTestCaseStatus(item.Status)
		if err != nil {
			return nil, err
		}

		results = append(results, &entity.SubmissionResult{
			SubmissionID:   submissionID,
			AttemptID:      attemptID,
			TestIndex:      item.Index,
			Status:         status,
			ActualOutput:   item.ActualOutput,
			Input:          item.Input,
			ExpectedOutput: item.ExpectedOutput,
			ExecutionTime:  item.ExecutionTime,
			MemoryUsed:     item.MemoryUsed,
		})
	}

	return results, nil
}

func mapTestCaseStatus(raw string) (entity.ResultStatus, error) {
	switch entity.ResultStatus(raw) {
	case entity.ResultAccepted,
		entity.ResultWrongAnswer,
		entity.ResultTimeLimit,
		entity.ResultMemoryLimit,
		entity.ResultRuntimeError,
		entity.ResultSystemError:
		return entity.ResultStatus(raw), nil
	default:
		return "", domain.ErrInvalidJudgeResult
	}
}
