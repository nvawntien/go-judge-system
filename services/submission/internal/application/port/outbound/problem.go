package outbound

import (
	"context"
	"go-judge-system/services/submission/internal/application/dto"
)

type ProblemReader interface {
	GetProblem(ctx context.Context, problemID int64, actor dto.ProblemActor) (dto.ProblemMetadata, error)
}
