package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type kafkaJudgePublisher struct {
	producer sarama.SyncProducer
	topic    string
	logger   *zap.Logger
}

func NewKafkaJudgePublisher(producer sarama.SyncProducer, kafkaCfg config.KafkaConfig, logger *zap.Logger) outbound.JudgePublisher {
	topic := kafkaCfg.JobTopic
	if topic == "" {
		topic = "judge.submission.jobs"
	}

	return &kafkaJudgePublisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

func (p *kafkaJudgePublisher) Publish(ctx context.Context, submission *entity.Submission) error {
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

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(strconv.FormatInt(submission.ID, 10)),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("publish judge job message: %w", err)
	}

	p.logger.Info(
		"published submission to judge queue",
		zap.Int64("submission_id", submission.ID),
		zap.String("attempt_id", payload.AttemptID),
		zap.String("topic", p.topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)
	return nil
}

