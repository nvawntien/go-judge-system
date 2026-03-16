package judge

import (
	"context"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"go.uber.org/zap"
)

type noopJudgePublisher struct {
	logger *zap.Logger
}

func NewNoopJudgePublisher(logger *zap.Logger) outbound.JudgePublisher {
	return &noopJudgePublisher{logger: logger}
}

func (p *noopJudgePublisher) Publish(_ context.Context, submission *entity.Submission) error {
	p.logger.Info(
		"judge publish skipped: noop publisher",
		zap.Int64("submission_id", submission.ID),
		zap.Int64("problem_id", submission.ProblemID),
	)
	return nil
}
