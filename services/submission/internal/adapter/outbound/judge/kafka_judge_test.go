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

	publisher := NewOutboxJudgePublisher(&mockOutboxRepository{}, config.KafkaConfig{})
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

	job := pkgjudge.JobMessage{
		SubmissionID: 77,
		ProblemID:    1001,
		ProblemSlug:  "two-sum",
		UserID:       "u-1",
		Language:     "GO",
		SourceCode:   "package main",
		AttemptID:    "attempt-77",
		EnqueuedAt:   time.Now().UTC(),
	}

	repo := &mockOutboxRepository{}
	repo.createFn = func(ctx context.Context, msg *entity.OutboxMessage) error {
		if msg.Topic != "judge.submission.jobs" {
			t.Fatalf("topic = %q, want %q", msg.Topic, "judge.submission.jobs")
		}

		if msg.AggregateID != job.SubmissionID {
			t.Fatalf("aggregate_id = %d, want %d", msg.AggregateID, job.SubmissionID)
		}

		var payload pkgjudge.JobMessage
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}

		if payload.SubmissionID != job.SubmissionID {
			t.Fatalf("submission_id = %d, want %d", payload.SubmissionID, job.SubmissionID)
		}
		if payload.ProblemID != job.ProblemID {
			t.Fatalf("problem_id = %d, want %d", payload.ProblemID, job.ProblemID)
		}
		if payload.ProblemSlug != "two-sum" {
			t.Fatalf("problem_slug = %q, want canonical slug", payload.ProblemSlug)
		}
		if payload.UserID != job.UserID {
			t.Fatalf("user_id = %q, want %q", payload.UserID, job.UserID)
		}
		if payload.Language != job.Language {
			t.Fatalf("language = %q, want %q", payload.Language, job.Language)
		}
		if payload.SourceCode != job.SourceCode {
			t.Fatalf("source_code = %q, want %q", payload.SourceCode, job.SourceCode)
		}
		if payload.AttemptID != "attempt-77" {
			t.Fatalf("attempt_id = %q, want persisted attempt", payload.AttemptID)
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
	)

	if err := publisher.Publish(context.Background(), job); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
}

func TestPublish_OutboxCreateError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockOutboxRepository{
		createFn: func(ctx context.Context, msg *entity.OutboxMessage) error {
			return wantErr
		},
	}

	publisher := NewOutboxJudgePublisher(repo, config.KafkaConfig{})

	err := publisher.Publish(context.Background(), pkgjudge.JobMessage{SubmissionID: 77, AttemptID: "attempt-77"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
}

func TestPublish_OmitsEmptyProblemSlug(t *testing.T) {
	t.Parallel()

	repo := &mockOutboxRepository{createFn: func(_ context.Context, msg *entity.OutboxMessage) error {
		var raw map[string]any
		if err := json.Unmarshal(msg.Payload, &raw); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if _, exists := raw["problem_slug"]; exists {
			t.Fatal("empty problem_slug must be omitted")
		}
		return nil
	}}

	publisher := NewOutboxJudgePublisher(repo, config.KafkaConfig{})
	if err := publisher.Publish(context.Background(), pkgjudge.JobMessage{SubmissionID: 77, AttemptID: "attempt-77"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestPublishRejectsEmptyAttemptID(t *testing.T) {
	t.Parallel()

	repo := &mockOutboxRepository{createFn: func(_ context.Context, _ *entity.OutboxMessage) error {
		t.Fatal("outbox repository must not be called for empty attempt")
		return nil
	}}
	publisher := NewOutboxJudgePublisher(repo, config.KafkaConfig{})

	if err := publisher.Publish(context.Background(), pkgjudge.JobMessage{SubmissionID: 77}); err == nil {
		t.Fatal("expected error for empty attempt ID")
	}
}
