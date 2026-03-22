package judge

import (
	"context"
	"fmt"

	"go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	"go.uber.org/zap"
)

type ProcessJudgeJobUseCase struct {
	executor        outbound.CodeExecutor
	resultPublisher outbound.ResultPublisher
	testCaseFetcher outbound.TestCaseFetcher
	logger          *zap.Logger
}

func NewProcessJudgeJobUseCase(
	executor outbound.CodeExecutor,
	resultPublisher outbound.ResultPublisher,
	testCaseFetcher outbound.TestCaseFetcher,
	logger *zap.Logger,
) *ProcessJudgeJobUseCase {
	return &ProcessJudgeJobUseCase{
		executor:        executor,
		resultPublisher: resultPublisher,
		testCaseFetcher: testCaseFetcher,
		logger:          logger,
	}
}

func (u *ProcessJudgeJobUseCase) Execute(ctx context.Context, jobMsg *judge.JobMessage) error {
	u.logger.Info(
		"processing judge job",
		zap.Int64("submission_id", jobMsg.SubmissionID),
		zap.Int64("problem_id", jobMsg.ProblemID),
		zap.String("attempt_id", jobMsg.AttemptID),
		zap.String("language", jobMsg.Language),
	)

	testCases, err := u.testCaseFetcher.FetchTestCases(ctx, jobMsg.ProblemID)
	if err != nil {
		u.logger.Error(
			"failed to fetch test cases",
			zap.Int64("submission_id", jobMsg.SubmissionID),
			zap.Int64("problem_id", jobMsg.ProblemID),
			zap.Error(err),
		)
		errMsg := fmt.Sprintf("fetch test cases error: %v", err)
		result := &outbound.ExecutionResult{
			Status: "SYSTEM_ERROR",
			Error:  &errMsg,
		}
		
		if pubErr := u.resultPublisher.PublishResult(ctx, jobMsg.SubmissionID, jobMsg.AttemptID, result); pubErr != nil {
			u.logger.Error("failed to publish system error result", zap.Error(pubErr))
		}
		
		return err
	}

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
