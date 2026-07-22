package problem

import (
	"context"
	"errors"

	inbound "go-judge-system/services/problem/internal/application/port/inbound/submission"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
)

type getProblemForSubmissionUseCase struct {
	problemRepo outbound.ProblemRepository
}

func NewGetProblemForSubmissionUseCase(problemRepo outbound.ProblemRepository) inbound.GetProblemForSubmissionUseCase {
	return &getProblemForSubmissionUseCase{problemRepo: problemRepo}
}

func (uc *getProblemForSubmissionUseCase) Execute(
	ctx context.Context,
	problemID int64,
) (inbound.GetProblemForSubmissionResult, error) {
	if problemID <= 0 {
		return inbound.GetProblemForSubmissionResult{}, domain.ErrInvalidInput
	}

	problem, err := uc.problemRepo.GetByID(ctx, problemID)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return inbound.GetProblemForSubmissionResult{}, err
		case errors.Is(err, domain.ErrProblemNotFound):
			return inbound.GetProblemForSubmissionResult{}, domain.ErrProblemNotFound
		default:
			return inbound.GetProblemForSubmissionResult{}, domain.ErrInternalServer.Wrap(err)
		}
	}
	if problem == nil {
		return inbound.GetProblemForSubmissionResult{}, domain.ErrInternalServer
	}
	if problem.IsHidden {
		return inbound.GetProblemForSubmissionResult{}, domain.ErrProblemNotFound
	}

	return inbound.GetProblemForSubmissionResult{
		ProblemID: problem.ID,
		Title:     problem.Title,
		Slug:      problem.TitleSlug,
	}, nil
}
