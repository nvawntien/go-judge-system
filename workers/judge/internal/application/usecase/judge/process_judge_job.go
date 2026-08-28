package judge

import (
	"context"
	"fmt"

	"go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"
	"go.uber.org/zap"
)

type ProcessJudgeJobUseCase struct {
	executor        outbound.CodeExecutor
	resultPublisher outbound.ResultPublisher
	metadataReader  outbound.ProblemTestCaseMetadataReader
	testCaseLoader  outbound.OfficialTestCaseLoader
	logger          *zap.Logger
}

func NewProcessJudgeJobUseCase(executor outbound.CodeExecutor, resultPublisher outbound.ResultPublisher,
	metadataReader outbound.ProblemTestCaseMetadataReader, testCaseLoader outbound.OfficialTestCaseLoader,
	logger *zap.Logger,
) *ProcessJudgeJobUseCase {
	return &ProcessJudgeJobUseCase{
		executor:        executor,
		resultPublisher: resultPublisher,
		metadataReader:  metadataReader,
		testCaseLoader:  testCaseLoader,
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

	metadata, err := u.metadataReader.GetTestCaseMetadata(ctx, jobMsg.ProblemID)
	if err != nil {
		return u.handleProcessingError(ctx, jobMsg, "fetch testcase metadata", err)
	}

	bundle, err := u.testCaseLoader.Load(ctx, metadata)
	if err != nil {
		return u.handleProcessingError(ctx, jobMsg, "load official testcases", err)
	}

	result, err := u.executor.Execute(ctx, outbound.ExecutionRequest{
		Language:           jobMsg.Language,
		SourceCode:         jobMsg.SourceCode,
		TestCases:          bundle.TestCases,
		StopOnFirstFailure: true,
		TestcaseDataset: &outbound.TestcaseDatasetIdentity{
			ProblemID:       metadata.ProblemID,
			Version:         bundle.Version,
			DatasetChecksum: bundle.DatasetChecksum,
		},
	})
	if err != nil {
		return u.handleProcessingError(ctx, jobMsg, "execute submission", err)
	}
	sanitizeOfficialResult(result)
	result.TestcaseVersion = intPtr(bundle.Version)
	result.TestCount = intPtr(bundle.TestCount)
	result.DatasetChecksum = nonEmptyStringPtr(bundle.DatasetChecksum)

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

func intPtr(value int) *int {
	return &value
}

func nonEmptyStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (u *ProcessJudgeJobUseCase) handleProcessingError(
	ctx context.Context,
	jobMsg *judge.JobMessage,
	stage string,
	err error,
) error {
	u.logger.Error(
		"judge job processing failed",
		zap.Int64("submission_id", jobMsg.SubmissionID),
		zap.Int64("problem_id", jobMsg.ProblemID),
		zap.String("attempt_id", jobMsg.AttemptID),
		zap.String("stage", stage),
		zap.Bool("non_retryable", workerdomain.IsNonRetryable(err)),
		zap.Error(err),
	)
	if !workerdomain.IsNonRetryable(err) {
		return err
	}

	errMsg := fmt.Sprintf("%s error: %v", stage, err)
	publicErrMsg := "The judge could not complete this submission."
	result := &outbound.ExecutionResult{
		Status:       "SYSTEM_ERROR",
		Error:        &errMsg,
		ErrorMessage: &publicErrMsg,
	}
	if pubErr := u.resultPublisher.PublishResult(ctx, jobMsg.SubmissionID, jobMsg.AttemptID, result); pubErr != nil {
		u.logger.Error("failed to publish non-retryable system error result", zap.Error(pubErr))
		return pubErr
	}
	return nil
}

func sanitizeOfficialResult(result *outbound.ExecutionResult) {
	if result == nil {
		return
	}
	for index := range result.TestCases {
		result.TestCases[index].Input = nil
		result.TestCases[index].ExpectedOutput = nil
	}
}
