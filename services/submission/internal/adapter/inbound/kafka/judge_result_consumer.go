package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/result"
	"go-judge-system/services/submission/internal/domain"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

const (
	defaultResultMaxRetries    = 3
	defaultResultRetryBaseWait = 500 * time.Millisecond
)

type JudgeResultConsumer struct {
	group        sarama.ConsumerGroup
	topic        string
	useCase      inbound.ApplyJudgeResultUseCase
	dltPublisher *DLTPublisher
	maxRetries   int
	logger       *zap.Logger
}

func NewJudgeResultConsumer(
	group sarama.ConsumerGroup,
	kafkaCfg config.KafkaConfig,
	useCase inbound.ApplyJudgeResultUseCase,
	dltPublisher *DLTPublisher,
	logger *zap.Logger,
) *JudgeResultConsumer {
	topic := strings.TrimSpace(kafkaCfg.ResultTopic)
	if topic == "" {
		topic = "judge.submission.results"
	}

	return &JudgeResultConsumer{
		group:        group,
		topic:        topic,
		useCase:      useCase,
		dltPublisher: dltPublisher,
		maxRetries:   defaultResultMaxRetries,
		logger:       logger,
	}
}

func (c *JudgeResultConsumer) Start(ctx context.Context) error {
	handler := &judgeResultHandler{
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

func (c *JudgeResultConsumer) Close() error {
	if c.group == nil {
		return nil
	}
	return c.group.Close()
}

type judgeResultHandler struct {
	useCase      inbound.ApplyJudgeResultUseCase
	dltPublisher *DLTPublisher
	maxRetries   int
	logger       *zap.Logger
}

func (h *judgeResultHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *judgeResultHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *judgeResultHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		select {
		case <-session.Context().Done():
			return nil
		default:
			h.handleMessage(session, msg)
		}
	}
	return nil
}

func (h *judgeResultHandler) handleMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	var payload pkgjudge.ResultMessage
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.Error("invalid judge result message - forwarding to DLT", zap.Int64("offset", msg.Offset), zap.Error(err))
		h.sendToDLT(session.Context(), msg, err.Error(), 0)
		session.MarkMessage(msg, "invalid_payload_dlt")
		return
	}

	var lastErr error
	for attempt := 1; attempt <= h.maxRetries; attempt++ {
		lastErr = h.useCase.Execute(session.Context(), payload)
		if lastErr == nil {
			session.MarkMessage(msg, "processed")
			return
		}

		if isNonRetryableJudgeResultError(lastErr) {
			h.logger.Warn(
				"non-retryable judge result message - forwarding to DLT",
				zap.Int64("submission_id", payload.SubmissionID),
				zap.String("attempt_id", payload.AttemptID),
				zap.Error(lastErr),
			)
			h.sendToDLT(session.Context(), msg, lastErr.Error(), attempt-1)
			session.MarkMessage(msg, "non_retryable_dlt")
			return
		}

		if session.Context().Err() != nil {
			h.logger.Warn("context cancelled during judge result retry, not committing offset", zap.Int64("submission_id", payload.SubmissionID))
			return
		}

		h.logger.Warn(
			"judge result processing failed, retrying",
			zap.Int64("submission_id", payload.SubmissionID),
			zap.String("attempt_id", payload.AttemptID),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", h.maxRetries),
			zap.Error(lastErr),
		)

		if attempt < h.maxRetries {
			backoff := defaultResultRetryBaseWait * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(backoff):
			case <-session.Context().Done():
				return
			}
		}
	}

	h.logger.Error(
		"judge result exceeded max retries - forwarding to DLT",
		zap.Int64("submission_id", payload.SubmissionID),
		zap.String("attempt_id", payload.AttemptID),
		zap.Int("max_retries", h.maxRetries),
		zap.Error(lastErr),
	)
	h.sendToDLT(session.Context(), msg, lastErr.Error(), h.maxRetries)
	session.MarkMessage(msg, "dlt_forwarded")
}

func (h *judgeResultHandler) sendToDLT(ctx context.Context, msg *sarama.ConsumerMessage, errMsg string, retryCount int) {
	if h.dltPublisher == nil {
		h.logger.Error("DLT publisher not configured, judge result message will be lost", zap.Int64("offset", msg.Offset))
		return
	}

	if err := h.dltPublisher.Publish(ctx, msg, errMsg, retryCount); err != nil {
		h.logger.Error("failed to publish judge result to DLT - message may be re-processed", zap.Int64("offset", msg.Offset), zap.Error(err))
	}
}

func isNonRetryableJudgeResultError(err error) bool {
	return errors.Is(err, domain.ErrInvalidJudgeResult) ||
		errors.Is(err, domain.ErrInvalidSubmissionStatus) ||
		errors.Is(err, domain.ErrSubmissionNotFound)
}
