package judge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/domain/entity"

	"go.uber.org/zap"
)

type mockOutboxRepository struct {
	createFn func(ctx context.Context, message *entity.OutboxMessage) error
}

func (m *mockOutboxRepository) Create(ctx context.Context, message *entity.OutboxMessage) error {
	if m.createFn != nil {
		return m.createFn(ctx, message)
	}
	message.ID = 1
	message.CreatedAt = time.Now()
	return nil
}

func (m *mockOutboxRepository) GetPending(ctx context.Context, limit int) ([]*entity.OutboxMessage, error) {
	return nil, nil
}

func (m *mockOutboxRepository) MarkPublished(ctx context.Context, id int64) error {
	return nil
}

func (m *mockOutboxRepository) MarkFailed(ctx context.Context, id int64, errReason string) error {
	return nil
}

func TestNewOutboxJudgePublisher_DefaultTopic(t *testing.T) {
	t.Parallel()

	publisher := NewOutboxJudgePublisher(&mockOutboxRepository{}, config.KafkaConfig{}, zap.NewNop())
	impl, ok := publisher.(*outboxJudgePublisher)
	if !ok {
		t.Fatal("expected outboxJudgePublisher implementation")
	}
	if impl.topic != "judge.submission.jobs" {
		t.Fatalf("topic = %q, want %q", impl.topic, "judge.submission.jobs")
	}
}

func TestPublish_Success(t *testing.T) {
	t.Parallel()

	sub := entity.NewSubmission(1001, "Two Sum", "u-1", "alice", entity.LanguageGo, "package main")
	sub.ID = 77

	repo := &mockOutboxRepository{}
	repo.createFn = func(ctx context.Context, msg *entity.OutboxMessage) error {
		if msg.Topic != "judge.submission.jobs" {
			t.Fatalf("topic = %q, want %q", msg.Topic, "judge.submission.jobs")
		}

		if msg.AggregateID != sub.ID {
			t.Fatalf("aggregate_id = %d, want %d", msg.AggregateID, sub.ID)
		}

		var payload pkgjudge.JobMessage
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
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
		if payload.AttemptID == "" {
			t.Fatal("attempt_id should not be empty")
		}
		if payload.EnqueuedAt.IsZero() {
			t.Fatal("enqueued_at should not be zero")
		}

		msg.ID = 1
		return nil
	}

	publisher := NewOutboxJudgePublisher(
		repo,
		config.KafkaConfig{JobTopic: "judge.submission.jobs"},
		zap.NewNop(),
	)

	if err := publisher.Publish(context.Background(), sub); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}

func TestPublish_OutboxCreateError(t *testing.T) {
	t.Parallel()

	sub := entity.NewSubmission(1001, "Two Sum", "u-1", "alice", entity.LanguageGo, "package main")
	sub.ID = 77

	wantErr := errors.New("db unavailable")
	repo := &mockOutboxRepository{
		createFn: func(ctx context.Context, msg *entity.OutboxMessage) error {
			return wantErr
		},
	}

	publisher := NewOutboxJudgePublisher(repo, config.KafkaConfig{}, zap.NewNop())

	err := publisher.Publish(context.Background(), sub)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
}
