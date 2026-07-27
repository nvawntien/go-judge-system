package outbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type relayOutboxRepository struct {
	getPendingFn    func(context.Context, int) ([]*entity.OutboxMessage, error)
	markPublishedFn func(context.Context, int64) error
	markFailedFn    func(context.Context, int64, string) error
}

func (r *relayOutboxRepository) Create(context.Context, *entity.OutboxMessage) error { return nil }

func (r *relayOutboxRepository) GetPending(ctx context.Context, limit int) ([]*entity.OutboxMessage, error) {
	if r.getPendingFn != nil {
		return r.getPendingFn(ctx, limit)
	}
	return nil, nil
}

func (r *relayOutboxRepository) MarkPublished(ctx context.Context, id int64) error {
	if r.markPublishedFn != nil {
		return r.markPublishedFn(ctx, id)
	}
	return nil
}

func (r *relayOutboxRepository) MarkFailed(ctx context.Context, id int64, reason string) error {
	if r.markFailedFn != nil {
		return r.markFailedFn(ctx, id, reason)
	}
	return nil
}

type relaySyncProducer struct {
	sendMessageFn func(*sarama.ProducerMessage) (int32, int64, error)
}

func (p *relaySyncProducer) SendMessage(message *sarama.ProducerMessage) (int32, int64, error) {
	return p.sendMessageFn(message)
}

func (p *relaySyncProducer) SendMessages([]*sarama.ProducerMessage) error { return nil }
func (p *relaySyncProducer) Close() error                                 { return nil }
func (p *relaySyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag      { return 0 }
func (p *relaySyncProducer) IsTransactional() bool                        { return false }
func (p *relaySyncProducer) BeginTxn() error                              { return nil }
func (p *relaySyncProducer) CommitTxn() error                             { return nil }
func (p *relaySyncProducer) AbortTxn() error                              { return nil }
func (p *relaySyncProducer) AddOffsetsToTxn(map[string][]*sarama.PartitionOffsetMetadata, string) error {
	return nil
}
func (p *relaySyncProducer) AddMessageToTxn(*sarama.ConsumerMessage, string, *string) error {
	return nil
}

func newObservedLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}

func testOutboxMessage(retryCount int) *entity.OutboxMessage {
	return &entity.OutboxMessage{
		ID:          17,
		AggregateID: 91,
		Topic:       "judge.submission.jobs",
		Payload:     []byte("sensitive-source-payload"),
		RetryCount:  retryCount,
	}
}

func requireStructuredFields(t *testing.T, entry observer.LoggedEntry, expected map[string]any) {
	t.Helper()
	fields := entry.ContextMap()
	for key, want := range expected {
		got, ok := fields[key]
		if !ok {
			t.Fatalf("log %q missing field %q: %#v", entry.Message, key, fields)
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("log %q field %q = %v, want %v", entry.Message, key, got, want)
		}
	}
	if _, ok := fields["payload"]; ok {
		t.Fatalf("log %q must not contain payload", entry.Message)
	}
	if _, ok := fields["source_code"]; ok {
		t.Fatalf("log %q must not contain source_code", entry.Message)
	}
	if strings.Contains(entry.Message, "sensitive-source-payload") {
		t.Fatalf("log message exposed payload: %q", entry.Message)
	}
}

func TestProcessPendingMessages_PublishAndMarkFailedErrorsAreLogged(t *testing.T) {
	publishErr := errors.New("broker unavailable")
	markFailedErr := errors.New("database unavailable")
	message := testOutboxMessage(2)
	markFailedCalls := 0
	repo := &relayOutboxRepository{
		getPendingFn: func(context.Context, int) ([]*entity.OutboxMessage, error) {
			return []*entity.OutboxMessage{message}, nil
		},
		markFailedFn: func(_ context.Context, id int64, reason string) error {
			markFailedCalls++
			if id != message.ID || reason != publishErr.Error() {
				t.Fatalf("MarkFailed(%d, %q)", id, reason)
			}
			return markFailedErr
		},
	}
	producer := &relaySyncProducer{sendMessageFn: func(*sarama.ProducerMessage) (int32, int64, error) {
		return 0, 0, publishErr
	}}
	logger, logs := newObservedLogger()
	relay := NewOutboxRelay(repo, producer, logger)

	relay.processPendingMessages(context.Background())

	if markFailedCalls != 1 {
		t.Fatalf("MarkFailed calls = %d, want 1", markFailedCalls)
	}
	for _, messageName := range []string{"failed to publish outbox message", "failed to persist outbox publish failure"} {
		entries := logs.FilterMessage(messageName).All()
		if len(entries) != 1 {
			t.Fatalf("log %q count = %d, want 1", messageName, len(entries))
		}
		if entries[0].Level != zap.ErrorLevel {
			t.Fatalf("log %q level = %s, want error", messageName, entries[0].Level)
		}
		requireStructuredFields(t, entries[0], map[string]any{
			"outbox_message_id": message.ID,
			"aggregate_id":      message.AggregateID,
			"topic":             message.Topic,
			"retry_count":       message.RetryCount,
		})
	}
	if logs.FilterMessage("outbox message reached retry limit").Len() != 0 {
		t.Fatal("retry-limit log must not be emitted when MarkFailed fails")
	}
}

func TestProcessPendingMessages_RetryLimitTransitionLoggedOnce(t *testing.T) {
	publishErr := errors.New("broker unavailable")
	beforeLimit := testOutboxMessage(maxPublishRetries - 1)
	atLimit := testOutboxMessage(maxPublishRetries)
	poll := 0
	markFailedCalls := 0
	repo := &relayOutboxRepository{
		getPendingFn: func(context.Context, int) ([]*entity.OutboxMessage, error) {
			poll++
			if poll == 1 {
				return []*entity.OutboxMessage{beforeLimit}, nil
			}
			return []*entity.OutboxMessage{atLimit}, nil
		},
		markFailedFn: func(context.Context, int64, string) error {
			markFailedCalls++
			return nil
		},
	}
	producer := &relaySyncProducer{sendMessageFn: func(*sarama.ProducerMessage) (int32, int64, error) {
		return 0, 0, publishErr
	}}
	logger, logs := newObservedLogger()
	relay := NewOutboxRelay(repo, producer, logger)

	relay.processPendingMessages(context.Background())
	relay.processPendingMessages(context.Background())

	if markFailedCalls != 1 {
		t.Fatalf("MarkFailed calls = %d, want 1", markFailedCalls)
	}
	entries := logs.FilterMessage("outbox message reached retry limit").All()
	if len(entries) != 1 {
		t.Fatalf("retry-limit log count = %d, want 1", len(entries))
	}
	requireStructuredFields(t, entries[0], map[string]any{
		"outbox_message_id": beforeLimit.ID,
		"aggregate_id":      beforeLimit.AggregateID,
		"topic":             beforeLimit.Topic,
		"retry_count":       maxPublishRetries,
	})
}

func TestProcessPendingMessages_SuccessMarksPublishedAndLogsDebug(t *testing.T) {
	message := testOutboxMessage(1)
	markPublishedCalls := 0
	repo := &relayOutboxRepository{
		getPendingFn: func(context.Context, int) ([]*entity.OutboxMessage, error) {
			return []*entity.OutboxMessage{message}, nil
		},
		markPublishedFn: func(_ context.Context, id int64) error {
			markPublishedCalls++
			if id != message.ID {
				t.Fatalf("MarkPublished ID = %d, want %d", id, message.ID)
			}
			return nil
		},
	}
	producer := &relaySyncProducer{sendMessageFn: func(kafkaMessage *sarama.ProducerMessage) (int32, int64, error) {
		if kafkaMessage.Topic != message.Topic {
			t.Fatalf("Kafka topic = %q, want %q", kafkaMessage.Topic, message.Topic)
		}
		return 3, 29, nil
	}}
	logger, logs := newObservedLogger()
	relay := NewOutboxRelay(repo, producer, logger)

	relay.processPendingMessages(context.Background())

	if markPublishedCalls != 1 {
		t.Fatalf("MarkPublished calls = %d, want 1", markPublishedCalls)
	}
	entries := logs.FilterMessage("published outbox message").All()
	if len(entries) != 1 {
		t.Fatalf("published log count = %d, want 1", len(entries))
	}
	if entries[0].Level != zap.DebugLevel {
		t.Fatalf("published log level = %s, want debug", entries[0].Level)
	}
	requireStructuredFields(t, entries[0], map[string]any{
		"outbox_message_id": message.ID,
		"aggregate_id":      message.AggregateID,
		"topic":             message.Topic,
		"retry_count":       message.RetryCount,
		"partition":         3,
		"offset":            29,
	})
}
