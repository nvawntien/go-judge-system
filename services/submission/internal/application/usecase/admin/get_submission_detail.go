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
	attemptRepo    outbound.SubmissionAttemptRepository
}

func NewGetAdminSubmissionDetailUseCase(
	submissionRepo outbound.SubmissionRepository,
	resultRepo outbound.SubmissionResultRepository,
	attemptRepos ...outbound.SubmissionAttemptRepository,
) inbound.GetAdminSubmissionDetailUseCase {
	var attemptRepo outbound.SubmissionAttemptRepository
	if len(attemptRepos) > 0 {
		attemptRepo = attemptRepos[0]
	}
	return &getAdminSubmissionDetailUseCase{
		submissionRepo: submissionRepo,
		resultRepo:     resultRepo,
		attemptRepo:    attemptRepo,
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

	var attempt *entity.SubmissionAttempt
	if uc.attemptRepo != nil && strings.TrimSpace(submission.CurrentAttemptID) != "" {
		attempt, err = uc.attemptRepo.GetByAttemptID(ctx, submission.CurrentAttemptID)
		if err != nil && !errors.Is(err, domain.ErrSubmissionNotFound) {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return dto.GetAdminSubmissionDetailResponse{}, err
			}
			return dto.GetAdminSubmissionDetailResponse{}, domain.ErrInternalServer.Wrap(err)
		}
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

	response := dto.GetAdminSubmissionDetailResponse{
		ID:                submission.ID,
		ProblemID:         submission.ProblemID,
		ProblemTitle:      submission.ProblemName,
		UserID:            submission.UserID,
		Username:          submission.Username,
		Language:          string(submission.Language),
		SourceCode:        submission.SourceCode,
		Status:            string(submission.Status),
		CurrentAttemptID:  submission.CurrentAttemptID,
		TotalTestCount:    knownTotalFromAttempt(attempt),
		TestcaseVersion:   testcaseVersionFromAttempt(attempt),
		DatasetChecksum:   datasetChecksumFromAttempt(attempt),
		PassedTestCount:   passed,
		ExecutedTestCount: len(testResults),
		RuntimeMS:         submission.ExecutionTime,
		MemoryKB:          submission.MemoryUsed,
		CompileMessage:    submission.CompileOutput,
		JudgeMessage:      submission.ErrorMessage,
		CreatedAt:         submission.CreatedAt,
		UpdatedAt:         submission.UpdatedAt,
		TestResults:       testResults,
	}
	if attempt != nil {
		trigger := string(attempt.TriggerType)
		createdAt := attempt.CreatedAt
		response.AttemptTrigger = &trigger
		response.AttemptTriggeredByUserID = attempt.TriggeredByUserID
		response.AttemptCreatedAt = &createdAt
	}
	return response, nil
}

func knownTotalFromAttempt(attempt *entity.SubmissionAttempt) *int {
	if attempt == nil {
		return nil
	}
	return attempt.TestCount
}

func testcaseVersionFromAttempt(attempt *entity.SubmissionAttempt) *int {
	if attempt == nil {
		return nil
	}
	return attempt.TestcaseVersion
}

func datasetChecksumFromAttempt(attempt *entity.SubmissionAttempt) *string {
	if attempt == nil {
		return nil
	}
	return attempt.DatasetChecksum
}
