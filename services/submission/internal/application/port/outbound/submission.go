package outbound

import (
	"context"
	"time"

	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/domain/entity"
)

type SubmissionRepository interface {
	Create(ctx context.Context, submission *entity.Submission) error
	GetByID(ctx context.Context, id int64) (*entity.Submission, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*entity.Submission, error)
	Update(ctx context.Context, submission *entity.Submission) error
	List(ctx context.Context, filter ListSubmissionsFilter) (ListSubmissionsResult, error)
	ResultSummaries(ctx context.Context, submissionIDs []int64) (map[int64]SubmissionResultSummary, error)
}

type SubmissionStreamSnapshotRepository interface {
	GetStreamSnapshot(ctx context.Context, submissionID int64) (*entity.SubmissionStreamSnapshot, error)
}

type ListSubmissionsFilter struct {
	UserID    *string
	Status    *string
	Language  *string
	ProblemID *int64
	Limit     int
	Offset    int
}

type ListSubmissionsResult struct {
	Items []*entity.Submission
	Total int64
}

type SubmissionResultSummary struct {
	SubmissionID int64
	Passed       int
	Total        int
}

type SubmissionResultRepository interface {
	GetBySubmissionID(ctx context.Context, submissionID int64) ([]*entity.SubmissionResult, error)
	DeleteBySubmissionID(ctx context.Context, submissionID int64) error
	ReplaceBySubmissionIDAndAttemptID(ctx context.Context, submissionID int64, attemptID string, results []*entity.SubmissionResult) error
}

type TransactionManager interface {
	ExecuteInTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type OutboxRepository interface {
	Create(ctx context.Context, message *entity.OutboxMessage) error
	GetPending(ctx context.Context, limit int) ([]*entity.OutboxMessage, error)
	MarkPublished(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, errReason string) error
}

type JudgePublisher interface {
	Publish(ctx context.Context, job pkgjudge.JobMessage) error
}

type AttemptIDGenerator interface {
	NewAttemptID() string
}

type SubmissionStreamTicketService interface {
	Issue(userID string, submissionID int64) (ticket string, expiresAt time.Time, err error)
	Verify(ticket string) (entity.SubmissionStreamTicketClaims, error)
}

type SubmissionEventHub interface {
	Subscribe(submissionID int64) (events <-chan entity.SubmissionEvent, unsubscribe func())
	Publish(event entity.SubmissionEvent)
}
