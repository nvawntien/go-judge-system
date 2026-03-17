package outbound

import (
	"context"

	"go-judge-system/services/submission/internal/domain/entity"
)

type SubmissionRepository interface {
	Create(ctx context.Context, submission *entity.Submission) error
	GetByID(ctx context.Context, id int64) (*entity.Submission, error)
	ListByUser(ctx context.Context, userID string, offset, limit int, status, language string) ([]*entity.Submission, error)
	CountByUser(ctx context.Context, userID string, status, language string) (int64, error)
	ListByProblem(ctx context.Context, problemID int64, offset, limit int, status, language string) ([]*entity.Submission, error)
	CountByProblem(ctx context.Context, problemID int64, status, language string) (int64, error)
	ListAll(ctx context.Context, offset, limit int, problemID *int64, userID, status, language string) ([]*entity.Submission, error)
	CountAll(ctx context.Context, problemID *int64, userID, status, language string) (int64, error)
}

type SubmissionResultRepository interface {
	GetBySubmissionID(ctx context.Context, submissionID int64) ([]*entity.SubmissionResult, error)
}

type JudgePublisher interface {
	Publish(ctx context.Context, submission *entity.Submission) error
}
