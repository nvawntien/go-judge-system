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
		SourceCode:  s.SourceCode,
		Status:      string(s.Status),
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
