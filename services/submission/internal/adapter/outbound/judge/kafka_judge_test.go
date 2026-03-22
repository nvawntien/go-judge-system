package judge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type mockSyncProducer struct {
	sendMessageFn func(msg *sarama.ProducerMessage) (int32, int64, error)
}

func (m *mockSyncProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	if m.sendMessageFn != nil {
		return m.sendMessageFn(msg)
	}
	return 0, 0, nil
}

func (m *mockSyncProducer) SendMessages(_ []*sarama.ProducerMessage) error {
	return nil
}

func (m *mockSyncProducer) Close() error {
	return nil
}

func (m *mockSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return 0
}

func (m *mockSyncProducer) IsTransactional() bool {
	return false
}

func (m *mockSyncProducer) BeginTxn() error {
	return nil
}

func (m *mockSyncProducer) CommitTxn() error {
	return nil
}

func (m *mockSyncProducer) AbortTxn() error {
	return nil
}

func (m *mockSyncProducer) AddOffsetsToTxn(_ map[string][]*sarama.PartitionOffsetMetadata, _ string) error {
	return nil
}

func (m *mockSyncProducer) AddMessageToTxn(_ *sarama.ConsumerMessage, _ string, _ *string) error {
	return nil
}

func TestNewKafkaJudgePublisher_DefaultTopic(t *testing.T) {
	t.Parallel()

	publisher := NewKafkaJudgePublisher(&mockSyncProducer{}, config.KafkaConfig{}, zap.NewNop())
	impl, ok := publisher.(*kafkaJudgePublisher)
	if !ok {
		t.Fatal("expected kafkaJudgePublisher implementation")
	}
	if impl.topic != "judge.submission.jobs" {
		t.Fatalf("topic = %q, want %q", impl.topic, "judge.submission.jobs")
	}
}

func TestPublish_Success(t *testing.T) {
	t.Parallel()

	sub := entity.NewSubmission(1001, "Two Sum", "u-1", "alice", entity.LanguageGo, "package main")
	sub.ID = 77

	producer := &mockSyncProducer{}
	producer.sendMessageFn = func(msg *sarama.ProducerMessage) (int32, int64, error) {
		if msg.Topic != "judge.submission.jobs" {
			t.Fatalf("topic = %q, want %q", msg.Topic, "judge.submission.jobs")
		}

		key, err := msg.Key.Encode()
		if err != nil {
			t.Fatalf("encode key: %v", err)
		}
		if string(key) != "77" {
			t.Fatalf("key = %q, want %q", string(key), "77")
		}

		val, err := msg.Value.Encode()
		if err != nil {
			t.Fatalf("encode value: %v", err)
		}

		var payload judgeJobMessage
		if err := json.Unmarshal(val, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}

		if payload.SubmissionID != sub.ID {
			t.Fatalf("submission_id = %d, want %d", payload.SubmissionID, sub.ID)
		}
		if payload.ProblemID != sub.ProblemID {
			t.Fatalf("problem_id = %d, want %d", payload.ProblemID, sub.ProblemID)
		}
		if payload.UserID != sub.UserID {
			t.Fatalf("user_id = %q, want %q", payload.UserID, sub.UserID)
		}
		if payload.Language != string(sub.Language) {
			t.Fatalf("language = %q, want %q", payload.Language, sub.Language)
		}
		if payload.SourceCode != sub.SourceCode {
			t.Fatalf("source_code = %q, want %q", payload.SourceCode, sub.SourceCode)
		}
		if payload.Status != string(sub.Status) {
			t.Fatalf("status = %q, want %q", payload.Status, sub.Status)
		}
		if payload.EnqueuedAt.IsZero() {
			t.Fatal("enqueued_at should not be zero")
		}

		return 2, 42, nil
	}

	publisher := NewKafkaJudgePublisher(
		producer,
		config.KafkaConfig{JobTopic: "judge.submission.jobs"},
		zap.NewNop(),
	)

	if err := publisher.Publish(context.Background(), sub); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}

func TestPublish_SendMessageError(t *testing.T) {
	t.Parallel()

	sub := entity.NewSubmission(1001, "Two Sum", "u-1", "alice", entity.LanguageGo, "package main")
	sub.ID = 77

	wantErr := errors.New("kafka unavailable")
	producer := &mockSyncProducer{
		sendMessageFn: func(_ *sarama.ProducerMessage) (int32, int64, error) {
			return 0, 0, wantErr
		},
	}

	publisher := NewKafkaJudgePublisher(producer, config.KafkaConfig{}, zap.NewNop())

	err := publisher.Publish(context.Background(), sub)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
}
