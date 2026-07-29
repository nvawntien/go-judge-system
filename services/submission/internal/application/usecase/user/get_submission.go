package user

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

type getSubmissionUseCase struct {
	submissionRepo outbound.SubmissionRepository
}

func NewGetSubmissionUseCase(submissionRepo outbound.SubmissionRepository) inbound.GetSubmissionUseCase {
	return &getSubmissionUseCase{submissionRepo: submissionRepo}
}

func (uc *getSubmissionUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.GetSubmissionRequest) (dto.GetSubmissionResponse, error) {
	if req.SubmissionID <= 0 {
		return dto.GetSubmissionResponse{}, domain.ErrInvalidSubmissionID
	}
	if claims.UserID == "" || claims.Role == "" {
		return dto.GetSubmissionResponse{}, domain.ErrSubmissionUnauthenticated
	}

	switch claims.Role {
	case rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin:
	default:
		return dto.GetSubmissionResponse{}, domain.ErrSubmissionForbidden
	}

	submission, err := uc.submissionRepo.GetByID(ctx, req.SubmissionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubmissionNotFound):
			return dto.GetSubmissionResponse{}, domain.ErrSubmissionNotFound
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return dto.GetSubmissionResponse{}, err
		default:
			return dto.GetSubmissionResponse{}, domain.ErrInternalServer.Wrap(err)
		}
	}
	if submission == nil {
		return dto.GetSubmissionResponse{}, domain.ErrInternalServer
	}

	if submission.UserID != claims.UserID &&
		claims.Role != rbac.RoleModerator &&
		claims.Role != rbac.RoleAdmin {
		return dto.GetSubmissionResponse{}, domain.ErrSubmissionNotFound
	}

	summaries, err := uc.submissionRepo.ResultSummaries(ctx, []int64{submission.ID})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return dto.GetSubmissionResponse{}, err
		}
		return dto.GetSubmissionResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	summary, found := summaries[submission.ID]
	passed, total := testcaseSummaryForStatus(submission.Status, summary, found)

	return dto.GetSubmissionResponse{
		ID:              submission.ID,
		ProblemID:       submission.ProblemID,
		ProblemTitle:    submission.ProblemName,
		UserID:          submission.UserID,
		Username:        submission.Username,
		Language:        string(submission.Language),
		SourceCode:      submission.SourceCode,
		Status:          string(submission.Status),
		ExecutionTimeMS: submission.ExecutionTime,
		MemoryUsedKB:    submission.MemoryUsed,
		PassedTestCases: passed,
		TotalTestCases:  total,
		CompileOutput:   submission.CompileOutput,
		ErrorMessage:    submission.ErrorMessage,
		CreatedAt:       submission.CreatedAt,
		UpdatedAt:       submission.UpdatedAt,
	}, nil
}
