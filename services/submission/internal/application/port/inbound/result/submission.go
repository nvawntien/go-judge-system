package result

import (
	"context"

	pkgjudge "go-judge-system/pkg/judge"
)

type ApplyJudgeResultUseCase interface {
	Execute(ctx context.Context, result pkgjudge.ResultMessage) error
}
