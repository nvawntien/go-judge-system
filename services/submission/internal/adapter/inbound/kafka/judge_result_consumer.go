package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type JudgeResultConsumer struct {
	group   sarama.ConsumerGroup
	topic   string
	useCase inbound.ProcessJudgeResultUseCase
	logger  *zap.Logger
}

func NewJudgeResultConsumer(
	group sarama.ConsumerGroup,
	kafkaCfg config.KafkaConfig,
	useCase inbound.ProcessJudgeResultUseCase,
	logger *zap.Logger,
) *JudgeResultConsumer {
	topic := strings.TrimSpace(kafkaCfg.ResultTopic)
	if topic == "" {
		topic = "judge.submission.results"
	}

	return &JudgeResultConsumer{
		group:   group,
		topic:   topic,
		useCase: useCase,
		logger:  logger,
	}
}

func (c *JudgeResultConsumer) Run(ctx context.Context) error {
	handler := &judgeResultHandler{useCase: c.useCase, logger: c.logger}

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

func (c *JudgeResultConsumer) Close() error {
	if c.group == nil {
		return nil
	}
	return c.group.Close()
}

type judgeResultHandler struct {
	useCase inbound.ProcessJudgeResultUseCase
	logger  *zap.Logger
}

func (h *judgeResultHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *judgeResultHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *judgeResultHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var payload dto.JudgeResultMessage
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.Error("invalid judge result message", zap.Error(err))
			session.MarkMessage(msg, "invalid_payload")
			continue
		}

		if err := h.useCase.Execute(session.Context(), payload); err != nil {
			h.logger.Error(
				"failed to process judge result",
				zap.Int64("submission_id", payload.SubmissionID),
				zap.Error(err),
			)
			continue
		}

		session.MarkMessage(msg, "processed")
	}

	return nil
}
