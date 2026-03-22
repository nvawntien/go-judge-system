package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/inbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

const (
	defaultMaxRetries    = 3
	defaultRetryBaseWait = 500 * time.Millisecond
)

type JudgeJobConsumer struct {
	group        sarama.ConsumerGroup
	topic        string
	useCase      inbound.ProcessJudgeJobUseCase
	dltPublisher *DLTPublisher
	maxRetries   int
	logger       *zap.Logger
}

func NewJudgeJobConsumer(
	group sarama.ConsumerGroup,
	kafkaCfg config.KafkaConfig,
	useCase inbound.ProcessJudgeJobUseCase,
	dltPublisher *DLTPublisher,
	logger *zap.Logger,
) *JudgeJobConsumer {
	topic := strings.TrimSpace(kafkaCfg.JobTopic)
	if topic == "" {
		topic = "judge.submission.jobs"
	}

	return &JudgeJobConsumer{
		group:        group,
		topic:        topic,
		useCase:      useCase,
		dltPublisher: dltPublisher,
		maxRetries:   defaultMaxRetries,
		logger:       logger,
	}
}

func (c *JudgeJobConsumer) Run(ctx context.Context) error {
	handler := &judgeJobHandler{
		useCase:      c.useCase,
		dltPublisher: c.dltPublisher,
		maxRetries:   c.maxRetries,
		logger:       c.logger,
	}

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

// -------------------------------------------------------------------
// Handler
// -------------------------------------------------------------------

type judgeJobHandler struct {
	useCase      inbound.ProcessJudgeJobUseCase
	dltPublisher *DLTPublisher
	maxRetries   int
	logger       *zap.Logger
}

func (h *judgeJobHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *judgeJobHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *judgeJobHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.handleMessage(session, msg)
	}
	return nil
}

func (h *judgeJobHandler) handleMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	// 1. Decode payload
	var payload judge.JobMessage
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.Error("invalid judge job message — forwarding to DLT",
			zap.Int64("offset", msg.Offset),
			zap.Error(err),
		)
		h.sendToDLT(session.Context(), msg, err.Error(), 0)
		session.MarkMessage(msg, "invalid_payload_dlt")
		return
	}

	// 2. Retry with exponential backoff
	var lastErr error
	for attempt := 1; attempt <= h.maxRetries; attempt++ {
		lastErr = h.useCase.Execute(session.Context(), &payload)
		if lastErr == nil {
			session.MarkMessage(msg, "processed")
			return
		}

		// If context is cancelled (shutdown), stop retrying immediately
		if session.Context().Err() != nil {
			h.logger.Warn("context cancelled during retry, not committing offset",
				zap.Int64("submission_id", payload.SubmissionID),
				zap.Int("attempt", attempt),
			)
			return
		}

		h.logger.Warn("judge job processing failed, retrying",
			zap.Int64("submission_id", payload.SubmissionID),
			zap.String("attempt_id", payload.AttemptID),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", h.maxRetries),
			zap.Error(lastErr),
		)

		if attempt < h.maxRetries {
			backoff := defaultRetryBaseWait * time.Duration(1<<(attempt-1)) // 500ms, 1s, 2s
			select {
			case <-time.After(backoff):
			case <-session.Context().Done():
				return
			}
		}
	}

	// 3. Max retries exhausted → send to DLT
	h.logger.Error("judge job exceeded max retries — forwarding to DLT",
		zap.Int64("submission_id", payload.SubmissionID),
		zap.String("attempt_id", payload.AttemptID),
		zap.Int("max_retries", h.maxRetries),
		zap.Error(lastErr),
	)
	h.sendToDLT(session.Context(), msg, lastErr.Error(), h.maxRetries)
	session.MarkMessage(msg, "dlt_forwarded")
}

func (h *judgeJobHandler) sendToDLT(ctx context.Context, msg *sarama.ConsumerMessage, errMsg string, retryCount int) {
	if h.dltPublisher == nil {
		h.logger.Error("DLT publisher not configured, message will be lost",
			zap.Int64("offset", msg.Offset),
		)
		return
	}

	if err := h.dltPublisher.Publish(ctx, msg, errMsg, retryCount); err != nil {
		h.logger.Error("failed to publish to DLT — message may be re-processed",
			zap.Int64("offset", msg.Offset),
			zap.Error(err),
		)
	}
}

