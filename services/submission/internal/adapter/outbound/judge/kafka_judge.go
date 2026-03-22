package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type outboxJudgePublisher struct {
	outboxRepo outbound.OutboxRepository
	topic      string
	logger     *zap.Logger
}

func NewOutboxJudgePublisher(outboxRepo outbound.OutboxRepository, kafkaCfg config.KafkaConfig, logger *zap.Logger) outbound.JudgePublisher {
	topic := kafkaCfg.JobTopic
	if topic == "" {
		topic = "judge.submission.jobs"
	}

	return &outboxJudgePublisher{
		outboxRepo: outboxRepo,
		topic:      topic,
		logger:     logger,
	}
}

func (p *outboxJudgePublisher) Publish(ctx context.Context, submission *entity.Submission) error {
	payload := pkgjudge.JobMessage{
		SubmissionID: submission.ID,
		ProblemID:    submission.ProblemID,
		ProblemSlug:  submission.ProblemName,
		UserID:       submission.UserID,
		Language:     string(submission.Language),
		SourceCode:   submission.SourceCode,
		AttemptID:    uuid.New().String(),
		EnqueuedAt:   time.Now().UTC(),
	}

	value, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal judge job payload: %w", err)
	}

	outboxMsg := &entity.OutboxMessage{
		AggregateID: submission.ID,
		Topic:       p.topic,
		Payload:     value,
		Status:      entity.OutboxStatusPending,
	}

	if err := p.outboxRepo.Create(ctx, outboxMsg); err != nil {
		return fmt.Errorf("create outbox message: %w", err)
	}

	p.logger.Info(
		"inserted judge job into outbox",
		zap.Int64("submission_id", submission.ID),
		zap.String("attempt_id", payload.AttemptID),
		zap.String("topic", p.topic),
		zap.Int64("outbox_id", outboxMsg.ID),
	)
	return nil
}
