package dto

import "time"

type CreateSubmissionRequest struct {
	ProblemID  int64  `json:"problem_id" binding:"required,gt=0"`
	Language   string `json:"language" binding:"required"`
	SourceCode string `json:"source_code" binding:"required"`
}

type CreateSubmissionResponse struct {
	ID           int64     `json:"id"`
	ProblemID    int64     `json:"problem_id"`
	ProblemTitle string    `json:"problem_title"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
