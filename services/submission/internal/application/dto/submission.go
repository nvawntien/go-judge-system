package dto

import (
	"time"
)

type CreateSubmissionRequest struct {
	ProblemID  int64  `json:"problem_id" binding:"required,gt=0"`
	Language   string `json:"language" binding:"required"`
	SourceCode string `json:"source_code" binding:"required"`
}

type RunCodeRequest struct {
	ProblemID  int64              `json:"problem_id" binding:"required,gt=0"`
	Language   string             `json:"language" binding:"required"`
	SourceCode string             `json:"source_code" binding:"required"`
	TestCases  []RunTestCaseInput `json:"testcases" binding:"required"`
}

type RunTestCaseInput struct {
	ID             string  `json:"id" binding:"required"`
	Kind           string  `json:"kind" binding:"required"`
	Stdin          string  `json:"stdin"`
	ExpectedOutput *string `json:"expected_output"`
}

type RunCodeResponse struct {
	Status        string              `json:"status"`
	CompileOutput string              `json:"compile_output"`
	Diagnostics   []CodeDiagnostic    `json:"diagnostics"`
	Tests         []RunTestCaseResult `json:"tests"`
}

const (
	RunCodeStatusCompleted        = "completed"
	RunCodeStatusCompilationError = "compile_error"
	RunCodeStatusSystemError      = "system_error"
)

type RunTestCaseResult struct {
	ID              string           `json:"id"`
	Kind            string           `json:"kind"`
	Status          string           `json:"status"`
	Stdout          string           `json:"stdout"`
	Stderr          string           `json:"stderr"`
	ExpectedOutput  *string          `json:"expected_output"`
	ExecutionTimeMS int64            `json:"execution_time_ms"`
	MemoryUsedKB    int64            `json:"memory_used_kb"`
	Diagnostics     []CodeDiagnostic `json:"diagnostics"`
}

type CodeDiagnostic struct {
	TestCaseID *string `json:"testcase_id,omitempty"`
	Kind       string  `json:"kind"`
	Severity   string  `json:"severity"`
	Message    string  `json:"message"`
	Line       int     `json:"line"`
	Column     int     `json:"column"`
	EndLine    *int    `json:"end_line,omitempty"`
	EndColumn  *int    `json:"end_column,omitempty"`
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
	ID              int64     `json:"id"`
	ProblemID       int64     `json:"problem_id"`
	ProblemTitle    string    `json:"problem_title"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	Language        string    `json:"language"`
	SourceCode      string    `json:"source_code"`
	Status          string    `json:"status"`
	ExecutionTimeMS *int      `json:"execution_time_ms"`
	MemoryUsedKB    *int      `json:"memory_used_kb"`
	PassedTestCases *int      `json:"passed_testcases"`
	TotalTestCases  *int      `json:"total_testcases"`
	CompileOutput   *string   `json:"compile_output"`
	ErrorMessage    *string   `json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
	ID              int64     `json:"id"`
	ProblemID       int64     `json:"problem_id"`
	ProblemTitle    string    `json:"problem_title"`
	Language        string    `json:"language"`
	Status          string    `json:"status"`
	ExecutionTimeMS *int      `json:"execution_time_ms"`
	MemoryUsedKB    *int      `json:"memory_used_kb"`
	PassedTestCases *int      `json:"passed_testcases"`
	TotalTestCases  *int      `json:"total_testcases"`
	CreatedAt       time.Time `json:"created_at"`
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
