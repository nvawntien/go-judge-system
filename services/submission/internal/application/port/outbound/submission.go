package outbound

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/submission/internal/domain/entity"
)

type SubmissionRepository interface {
	Create(ctx context.Context, submission *entity.Submission) error
	GetByID(ctx context.Context, id int64) (*entity.Submission, error)
	Update(ctx context.Context, submission *entity.Submission) error
	ListByUser(ctx context.Context, userID string, offset, limit int, status, language string) ([]*entity.Submission, error)
	CountByUser(ctx context.Context, userID string, status, language string) (int64, error)
	ListByProblem(ctx context.Context, problemID int64, offset, limit int, status, language string) ([]*entity.Submission, error)
	CountByProblem(ctx context.Context, problemID int64, status, language string) (int64, error)
	ListAll(ctx context.Context, offset, limit int, problemID *int64, userID, status, language string) ([]*entity.Submission, error)
	CountAll(ctx context.Context, problemID *int64, userID, status, language string) (int64, error)
}

type SubmissionResultRepository interface {
	GetBySubmissionID(ctx context.Context, submissionID int64) ([]*entity.SubmissionResult, error)
	DeleteBySubmissionID(ctx context.Context, submissionID int64) error
	ReplaceBySubmissionID(ctx context.Context, submissionID int64, results []*entity.SubmissionResult) error
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

type ProblemAccessChecker interface {
	CanManageProblem(ctx context.Context, claims auth.Claims, problemID int64) (bool, error)
}

type JudgePublisher interface {
	Publish(ctx context.Context, submission *entity.Submission) error
}
