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
)

type outboxJudgePublisher struct {
	outboxRepo outbound.OutboxRepository
	topic      string
}

func NewOutboxJudgePublisher(outboxRepo outbound.OutboxRepository, kafkaCfg config.KafkaConfig) outbound.JudgePublisher {
	topic := kafkaCfg.JobTopic
	if topic == "" {
		topic = "judge.submission.jobs"
	}

	return &outboxJudgePublisher{
		outboxRepo: outboxRepo,
		topic:      topic,
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
		return err
	}

	return nil
}
