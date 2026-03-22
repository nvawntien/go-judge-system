package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/inbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type JudgeJobConsumer struct {
	group   sarama.ConsumerGroup
	topic   string
	useCase inbound.ProcessJudgeJobUseCase
	logger  *zap.Logger
}

func NewJudgeJobConsumer(
	group sarama.ConsumerGroup,
	kafkaCfg config.KafkaConfig,
	useCase inbound.ProcessJudgeJobUseCase,
	logger *zap.Logger,
) *JudgeJobConsumer {
	topic := strings.TrimSpace(kafkaCfg.JobTopic)
	if topic == "" {
		topic = "judge.submission.jobs"
	}

	return &JudgeJobConsumer{
		group:   group,
		topic:   topic,
		useCase: useCase,
		logger:  logger,
	}
}

func (c *JudgeJobConsumer) Run(ctx context.Context) error {
	handler := &judgeJobHandler{useCase: c.useCase, logger: c.logger}

	for {
		if err := c.group.Consume(ctx, []string{c.topic}, handler); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		if ctx.Err() != nil {
			return nil
		}
	}
}

func (c *JudgeJobConsumer) Close() error {
	if c.group == nil {
		return nil
	}
	return c.group.Close()
}

type judgeJobHandler struct {
	useCase inbound.ProcessJudgeJobUseCase
	logger  *zap.Logger
}

func (h *judgeJobHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *judgeJobHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *judgeJobHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var payload judge.JobMessage
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.Error("invalid judge job message", zap.Error(err))
			session.MarkMessage(msg, "invalid_payload")
			continue
		}

		if err := h.useCase.Execute(session.Context(), &payload); err != nil {
			h.logger.Error(
				"failed to process judge job",
				zap.Int64("submission_id", payload.SubmissionID),
				zap.Error(err),
			)
			continue
		}

		session.MarkMessage(msg, "processed")
	}

	return nil
}
