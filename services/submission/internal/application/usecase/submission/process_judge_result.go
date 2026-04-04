package submission

import (
	"context"
	"fmt"

	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"go.uber.org/zap"
)

type processJudgeResultUseCase struct {
	submissionRepo       outbound.SubmissionRepository
	submissionResultRepo outbound.SubmissionResultRepository
	logger               *zap.Logger
}

func NewProcessJudgeResultUseCase(
	submissionRepo outbound.SubmissionRepository,
	submissionResultRepo outbound.SubmissionResultRepository,
	logger *zap.Logger,
) inbound.ProcessJudgeResultUseCase {
	return &processJudgeResultUseCase{
		submissionRepo:       submissionRepo,
		submissionResultRepo: submissionResultRepo,
		logger:               logger,
	}
}

func (uc *processJudgeResultUseCase) Execute(ctx context.Context, message dto.JudgeResultMessage) error {
	submission, err := uc.submissionRepo.GetByID(ctx, message.SubmissionID)
	if err != nil {
		return fmt.Errorf("load submission: %w", err)
	}

	status, err := parseSubmissionStatus(message.Status)
	if err != nil {
		return err
	}

	submission.MarkCompleted(status, message.ExecutionTime, message.MemoryUsed, message.CompileOutput)
	if err := uc.submissionRepo.Update(ctx, submission); err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}

	results := make([]*entity.SubmissionResult, 0, len(message.Results))
	for _, item := range message.Results {
		itemStatus, err := parseResultStatus(item.Status)
		if err != nil {
			return err
		}

		results = append(results, &entity.SubmissionResult{
			SubmissionID:  submission.ID,
			TestIndex:     item.Index,
			Status:        itemStatus,
			ActualOutput:  item.ActualOutput,
			ExecutionTime: item.ExecutionTime,
			MemoryUsed:    item.MemoryUsed,
		})
	}

	if err := uc.submissionResultRepo.ReplaceBySubmissionID(ctx, submission.ID, results); err != nil {
		return fmt.Errorf("replace submission results: %w", err)
	}

	uc.logger.Info(
		"judge result applied",
		zap.Int64("submission_id", submission.ID),
		zap.String("status", message.Status),
		zap.Int("result_count", len(results)),
	)
	return nil
}

func parseSubmissionStatus(raw string) (entity.Status, error) {
	switch entity.Status(raw) {
	case entity.StatusAccepted,
		entity.StatusWrongAnswer,
		entity.StatusTimeLimitExceed,
		entity.StatusMemoryLimitExceed,
		entity.StatusRuntimeError,
		entity.StatusCompilationError,
		entity.StatusSystemError:
		return entity.Status(raw), nil
	default:
		return "", fmt.Errorf("invalid judge overall status: %s", raw)
	}
}

func parseResultStatus(raw string) (entity.ResultStatus, error) {
	switch entity.ResultStatus(raw) {
	case entity.ResultAccepted,
		entity.ResultWrongAnswer,
		entity.ResultTimeLimit,
		entity.ResultMemoryLimit,
		entity.ResultRuntimeError:
		return entity.ResultStatus(raw), nil
	default:
		return "", fmt.Errorf("invalid judge testcase status: %s", raw)
	}
}
