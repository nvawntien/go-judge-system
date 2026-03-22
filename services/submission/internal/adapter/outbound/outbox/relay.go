package outbox

import (
	"context"
	"strconv"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type OutboxRelay struct {
	repo     outbound.OutboxRepository
	producer sarama.SyncProducer
	logger   *zap.Logger
}

func NewOutboxRelay(repo outbound.OutboxRepository, producer sarama.SyncProducer, logger *zap.Logger) *OutboxRelay {
	return &OutboxRelay{
		repo:     repo,
		producer: producer,
		logger:   logger,
	}
}

func (r *OutboxRelay) Start(ctx context.Context, pollInterval time.Duration) error {
	r.logger.Info("starting outbox relay", zap.Duration("interval", pollInterval))
	
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopping outbox relay")
			return nil
		case <-ticker.C:
			r.processPendingMessages(ctx)
		}
	}
}

func (r *OutboxRelay) processPendingMessages(ctx context.Context) {
	// Fetch up to 100 pending messages at a time
	messages, err := r.repo.GetPending(ctx, 100)
	if err != nil {
		r.logger.Error("failed to get pending outbox messages", zap.Error(err))
		return
	}

	for _, msg := range messages {
		// Stop processing if context is cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Skip messages that have failed more than 5 times
		if msg.RetryCount >= 5 {
			continue // In a real system you'd move this to a DLQ or trigger alerts
		}

		kafkaMsg := &sarama.ProducerMessage{
			Topic: msg.Topic,
			Key:   sarama.StringEncoder(strconv.FormatInt(msg.AggregateID, 10)),
			Value: sarama.ByteEncoder(msg.Payload),
		}

		partition, offset, err := r.producer.SendMessage(kafkaMsg)
		if err != nil {
			r.logger.Error("failed to publish outbox message to kafka",
				zap.Int64("outbox_id", msg.ID),
				zap.Error(err),
			)
			_ = r.repo.MarkFailed(ctx, msg.ID, err.Error())
			continue
		}

		// Mark as published on success
		if err := r.repo.MarkPublished(ctx, msg.ID); err != nil {
			r.logger.Error("failed to mark outbox message as published",
				zap.Int64("outbox_id", msg.ID),
				zap.Error(err),
			)
			// At this point we might have published to Kafka but failed to update status in DB.
			// The transaction outbox pattern naturally handles this by guaranteeing at-least-once delivery.
			// The consumer MUST handle idempotency (which we implemented via AttemptID in Worker Phase 2).
		} else {
			r.logger.Debug("published outbox message",
				zap.Int64("outbox_id", msg.ID),
				zap.String("topic", msg.Topic),
				zap.Int32("partition", partition),
				zap.Int64("offset", offset),
			)
		}
	}
}
