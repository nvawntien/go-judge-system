package outbound

import (
	"context"

	"go-judge-system/pkg/rbac"
)

type ProblemActor struct {
	UserID string
	Role   rbac.Role
}

type ProblemMetadata struct {
	ID    int64
	Title string
	Slug  string
}

type ProblemReader interface {
	GetProblem(ctx context.Context, problemID int64, actor ProblemActor) (ProblemMetadata, error)
}
