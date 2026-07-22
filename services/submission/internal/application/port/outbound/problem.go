package outbound

import "context"

type ProblemForSubmission struct {
	ID    int64
	Title string
	Slug  string
}

type ProblemReader interface {
	GetForSubmission(ctx context.Context, problemID int64) (ProblemForSubmission, error)
}
