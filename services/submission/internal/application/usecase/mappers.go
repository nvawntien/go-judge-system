package usecase

import (
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain/entity"
)

func MapSubmissionToResponse(s *entity.Submission) dto.SubmissionResponse {
	return dto.SubmissionResponse{
		ID:          s.ID,
		ProblemID:   s.ProblemID,
		ProblemName: s.ProblemName,
		UserID:      s.UserID,
		Username:    s.Username,
		Language:    string(s.Language),
		Status:      string(s.Status),
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func MapSubmissionToDetailResponse(s *entity.Submission, results []*entity.SubmissionResult) dto.SubmissionDetailResponse {
	resp := dto.SubmissionDetailResponse{
		SubmissionResponse: MapSubmissionToResponse(s),
		SourceCode:         s.SourceCode,
		ExecutionTimeMs:    s.ExecutionTime,
		MemoryUsedKB:       s.MemoryUsed,
		CompileOutput:      s.CompileOutput,
		TotalTests:         len(results),
	}

	// For non-ACCEPTED results: find the first failed test case
	if s.Status != entity.StatusAccepted && s.Status != entity.StatusPending && s.Status != entity.StatusJudging {
		for _, r := range results {
			if r.Status != entity.ResultAccepted {
				idx := r.TestIndex
				resp.FailedTestIndex = &idx
				resp.FailedTest = &dto.SubmissionResultResponse{
					TestIndex:      r.TestIndex,
					Status:         string(r.Status),
					Input:          r.Input,
					ExpectedOutput: r.ExpectedOutput,
					ActualOutput:   r.ActualOutput,
					ExecutionTime:  r.ExecutionTime,
					MemoryUsed:     r.MemoryUsed,
				}
				break
			}
		}
	}

	return resp
}
