package admin

import (
	"context"
	"errors"
	"sort"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/admin"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type getAdminSubmissionDetailUseCase struct {
	submissionRepo outbound.SubmissionRepository
	resultRepo     outbound.SubmissionResultRepository
}

func NewGetAdminSubmissionDetailUseCase(
	submissionRepo outbound.SubmissionRepository,
	resultRepo outbound.SubmissionResultRepository,
) inbound.GetAdminSubmissionDetailUseCase {
	return &getAdminSubmissionDetailUseCase{
		submissionRepo: submissionRepo,
		resultRepo:     resultRepo,
	}
}

func (uc *getAdminSubmissionDetailUseCase) Execute(
	ctx context.Context,
	claims auth.Claims,
	req dto.GetAdminSubmissionDetailRequest,
) (dto.GetAdminSubmissionDetailResponse, error) {
	if strings.TrimSpace(claims.UserID) == "" || claims.Role == "" {
		return dto.GetAdminSubmissionDetailResponse{}, domain.ErrSubmissionUnauthenticated
	}
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.GetAdminSubmissionDetailResponse{}, domain.ErrSubmissionForbidden
	}
	if req.SubmissionID <= 0 {
		return dto.GetAdminSubmissionDetailResponse{}, domain.ErrInvalidSubmissionID
	}

	submission, err := uc.submissionRepo.GetByID(ctx, req.SubmissionID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, domain.ErrSubmissionNotFound) {
			return dto.GetAdminSubmissionDetailResponse{}, err
		}
		return dto.GetAdminSubmissionDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	if submission == nil {
		return dto.GetAdminSubmissionDetailResponse{}, domain.ErrSubmissionNotFound
	}

	results, err := uc.resultRepo.GetBySubmissionIDAndAttemptID(
		ctx,
		submission.ID,
		submission.CurrentAttemptID,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return dto.GetAdminSubmissionDetailResponse{}, err
		}
		return dto.GetAdminSubmissionDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i] == nil {
			return false
		}
		if results[j] == nil {
			return true
		}
		return results[i].TestIndex < results[j].TestIndex
	})

	testResults := make([]dto.AdminSubmissionTestResult, 0, len(results))
	passed := 0
	for _, result := range results {
		if result == nil {
			return dto.GetAdminSubmissionDetailResponse{}, domain.ErrInternalServer
		}
		if result.Status == entity.ResultAccepted {
			passed++
		}
		testResults = append(testResults, dto.AdminSubmissionTestResult{
			Index:     result.TestIndex,
			Status:    string(result.Status),
			RuntimeMS: result.ExecutionTime,
			MemoryKB:  result.MemoryUsed,
		})
	}

	return dto.GetAdminSubmissionDetailResponse{
		ID:                submission.ID,
		ProblemID:         submission.ProblemID,
		ProblemTitle:      submission.ProblemName,
		UserID:            submission.UserID,
		Username:          submission.Username,
		Language:          string(submission.Language),
		SourceCode:        submission.SourceCode,
		Status:            string(submission.Status),
		CurrentAttemptID:  submission.CurrentAttemptID,
		PassedTestCount:   passed,
		ExecutedTestCount: len(testResults),
		TotalTestCount:    inferKnownTotalTestCount(submission.Status, len(testResults)),
		RuntimeMS:         submission.ExecutionTime,
		MemoryKB:          submission.MemoryUsed,
		CompileMessage:    submission.CompileOutput,
		JudgeMessage:      submission.ErrorMessage,
		CreatedAt:         submission.CreatedAt,
		UpdatedAt:         submission.UpdatedAt,
		TestResults:       testResults,
	}, nil
}

func inferKnownTotalTestCount(status entity.Status, executed int) *int {
	switch {
	case status == entity.StatusAccepted:
		return intValue(executed)
	case executed == 0 && status == entity.StatusCompilationError:
		return intValue(0)
	default:
		return nil
	}
}

func intValue(value int) *int {
	return &value
}
