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

func MapSubmissionResultToResponse(r *entity.SubmissionResult) dto.SubmissionResultResponse {
	return dto.SubmissionResultResponse{
		ID:            r.ID,
		TestCaseID:    r.TestCaseID,
		Status:        string(r.Status),
		ActualOutput:  r.ActualOutput,
		ExecutionTime: r.ExecutionTime,
		MemoryUsed:    r.MemoryUsed,
		Order:         r.Order,
	}
}

func MapSubmissionToDetailResponse(s *entity.Submission, results []*entity.SubmissionResult) dto.SubmissionDetailResponse {
	res := make([]dto.SubmissionResultResponse, 0, len(results))
	for _, r := range results {
		res = append(res, MapSubmissionResultToResponse(r))
	}

	return dto.SubmissionDetailResponse{
		SubmissionResponse: MapSubmissionToResponse(s),
		SourceCode:         s.SourceCode,
		ExecutionTime:      s.ExecutionTime,
		MemoryUsed:         s.MemoryUsed,
		CompileOutput:      s.CompileOutput,
		Results:            res,
	}
}
