package outbound

import (
	"context"
	"time"

	"go-judge-system/services/problem/internal/domain/entity"
)

// SubmissionProblemMetadata is the canonical state required to authorize a
// submission and populate the existing Problem gRPC response. It deliberately
// excludes statement content, tags, examples, hints, and testcase data.
type SubmissionProblemMetadata struct {
	ID          int64
	Title       string
	Slug        string
	TimeLimit   float64
	MemoryLimit int
	AuthorID    string
	IsHidden    bool
}

// SubmissionProblemRepository is a narrow read model for the synchronous
// submission eligibility path. PostgreSQL remains the source of truth.
type SubmissionProblemRepository interface {
	GetSubmissionProblem(ctx context.Context, id int64) (SubmissionProblemMetadata, error)
}

// SubmissionProblemCache accelerates canonical problem-state reads. It never
// stores actor-specific authorization decisions.
type SubmissionProblemCache interface {
	Get(ctx context.Context, problemID int64) (SubmissionProblemMetadata, bool, error)
	Set(ctx context.Context, metadata SubmissionProblemMetadata, ttl time.Duration) error
	Delete(ctx context.Context, problemID int64) error
}

type ProblemRepository interface {
	Create(ctx context.Context, problem *entity.Problem) error
	GetByID(ctx context.Context, id int64) (*entity.Problem, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Problem, error)
	Update(ctx context.Context, problem *entity.Problem) error
	Delete(ctx context.Context, id int64) error // soft delete
	List(ctx context.Context, offset, limit int, difficulty, search, tagSlug string, includeHidden bool) ([]*entity.Problem, error)
	Count(ctx context.Context, difficulty, search, tagSlug string, includeHidden bool) (int64, error)
	ListByAuthor(ctx context.Context, authorID string, offset, limit int, difficulty, search, tagSlug string) ([]*entity.Problem, error)
	CountByAuthor(ctx context.Context, authorID string, difficulty, search, tagSlug string) (int64, error)
	ListForFormatBackfill(ctx context.Context) ([]*entity.Problem, error)
	UpdateFormatsForBackfill(ctx context.Context, id int64, description, inputFormat, outputFormat string) (bool, error)
}
