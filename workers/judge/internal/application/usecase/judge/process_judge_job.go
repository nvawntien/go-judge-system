package judge

import (
	"context"
	"fmt"

	"go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	"go.uber.org/zap"
)

type ProcessJudgeJobUseCase struct {
	executor       outbound.CodeExecutor
	resultPublisher outbound.ResultPublisher
	logger         *zap.Logger
}

func NewProcessJudgeJobUseCase(
	executor outbound.CodeExecutor,
	resultPublisher outbound.ResultPublisher,
	logger *zap.Logger,
) *ProcessJudgeJobUseCase {
	return &ProcessJudgeJobUseCase{
		executor:        executor,
		resultPublisher: resultPublisher,
		logger:          logger,
	}
}

func (u *ProcessJudgeJobUseCase) Execute(ctx context.Context, jobMsg *judge.JobMessage) error {
	u.logger.Info(
		"processing judge job",
		zap.Int64("submission_id", jobMsg.SubmissionID),
		zap.Int64("problem_id", jobMsg.ProblemID),
		zap.String("language", jobMsg.Language),
	)

	// TODO: Fetch test cases from problem service
	// For now, use empty test cases
	testCases := []outbound.TestCase{}

	// Execute code
	result, err := u.executor.Execute(ctx, jobMsg.Language, jobMsg.SourceCode, testCases)
	if err != nil {
		u.logger.Error(
			"code execution failed",
			zap.Int64("submission_id", jobMsg.SubmissionID),
			zap.Error(err),
		)
		errMsg := fmt.Sprintf("execution error: %v", err)
		result = &outbound.ExecutionResult{
			Status: "RUNTIME_ERROR",
			Error:  &errMsg,
		}
	}

	// Publish result
	if err := u.resultPublisher.PublishResult(ctx, jobMsg.SubmissionID, jobMsg.AttemptID, result); err != nil {
		u.logger.Error(
			"failed to publish judge result",
			zap.Int64("submission_id", jobMsg.SubmissionID),
			zap.Error(err),
		)
		return err
	}

	u.logger.Info(
		"judge job completed",
		zap.Int64("submission_id", jobMsg.SubmissionID),
		zap.String("status", result.Status),
	)

	return nil
}
