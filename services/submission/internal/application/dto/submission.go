package dto

import (
	"time"
)

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

type GetSubmissionRequest struct {
	SubmissionID int64 `uri:"submission_id" binding:"required,gt=0"`
}

type GetSubmissionResponse struct {
	ID           int64     `json:"id"`
	ProblemID    int64     `json:"problem_id"`
	ProblemTitle string    `json:"problem_title"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Language     string    `json:"language"`
	SourceCode   string    `json:"source_code"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListMySubmissionsRequest struct {
	Page      *int   `form:"page"`
	Limit     *int   `form:"limit"`
	Status    string `form:"status"`
	Language  string `form:"language"`
	ProblemID *int64 `form:"problem_id"`
}

type ListAdminSubmissionsRequest struct {
	Page      *int    `form:"page"`
	Limit     *int    `form:"limit"`
	Status    *string `form:"status"`
	Language  *string `form:"language"`
	ProblemID *int64  `form:"problem_id"`
	UserID    *string `form:"user_id"`
}

type SubmissionListItem struct {
	ID           int64     `json:"id"`
	ProblemID    int64     `json:"problem_id"`
	ProblemTitle string    `json:"problem_title"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type ListMySubmissionsResponse struct {
	Items      []SubmissionListItem `json:"items"`
	Pagination PaginationResponse   `json:"pagination"`
}

type AdminSubmissionListItem struct {
	ID           int64     `json:"id"`
	ProblemID    int64     `json:"problem_id"`
	ProblemTitle string    `json:"problem_title"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type ListAdminSubmissionsResponse struct {
	Items      []AdminSubmissionListItem `json:"items"`
	Pagination PaginationResponse        `json:"pagination"`
}
