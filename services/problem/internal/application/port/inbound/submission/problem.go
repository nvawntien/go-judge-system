package submission

import "context"

type GetProblemForSubmissionResult struct {
	ProblemID int64
	Title     string
	Slug      string
}

type GetProblemForSubmissionUseCase interface {
	Execute(ctx context.Context, problemID int64) (GetProblemForSubmissionResult, error)
}
