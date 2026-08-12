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

type IssueSubmissionStreamTicketRequest struct {
	SubmissionID int64 `uri:"submission_id" binding:"required,gt=0"`
}

type IssueSubmissionStreamTicketResponse struct {
	Ticket    string    `json:"ticket"`
	ExpiresAt time.Time `json:"expires_at"`
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

type GetAdminSubmissionDetailRequest struct {
	SubmissionID int64 `uri:"submission_id" binding:"required,gt=0"`
}

type RejudgeAdminSubmissionRequest struct {
	SubmissionID int64 `uri:"submission_id" binding:"required,gt=0"`
}

type RejudgeAdminSubmissionResponse struct {
	SubmissionID             int64     `json:"submission_id"`
	AttemptID                string    `json:"attempt_id"`
	Status                   string    `json:"status"`
	AttemptTrigger           string    `json:"attempt_trigger"`
	AttemptTriggeredByUserID string    `json:"attempt_triggered_by_user_id"`
	EnqueuedAt               time.Time `json:"enqueued_at"`
}

type AdminSubmissionTestResult struct {
	Index     int    `json:"index"`
	Status    string `json:"status"`
	RuntimeMS *int   `json:"runtime_ms"`
	MemoryKB  *int   `json:"memory_kb"`
}

type GetAdminSubmissionDetailResponse struct {
	ID                       int64                       `json:"id"`
	ProblemID                int64                       `json:"problem_id"`
	ProblemTitle             string                      `json:"problem_title"`
	UserID                   string                      `json:"user_id"`
	Username                 string                      `json:"username"`
	Language                 string                      `json:"language"`
	SourceCode               string                      `json:"source_code"`
	Status                   string                      `json:"status"`
	CurrentAttemptID         string                      `json:"current_attempt_id"`
	AttemptTrigger           *string                     `json:"attempt_trigger"`
	AttemptTriggeredByUserID *string                     `json:"attempt_triggered_by_user_id"`
	AttemptCreatedAt         *time.Time                  `json:"attempt_created_at"`
	TestcaseVersion          *int                        `json:"testcase_version"`
	DatasetChecksum          *string                     `json:"dataset_checksum"`
	PassedTestCount          int                         `json:"passed_test_count"`
	ExecutedTestCount        int                         `json:"executed_test_count"`
	TotalTestCount           *int                        `json:"total_test_count"`
	RuntimeMS                *int                        `json:"runtime_ms"`
	MemoryKB                 *int                        `json:"memory_kb"`
	CompileMessage           *string                     `json:"compile_message"`
	JudgeMessage             *string                     `json:"judge_message"`
	CreatedAt                time.Time                   `json:"created_at"`
	UpdatedAt                time.Time                   `json:"updated_at"`
	TestResults              []AdminSubmissionTestResult `json:"test_results"`
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

type GetMyProfileStatsResponse struct {
	TotalSubmissions     int64                          `json:"total_submissions"`
	AttemptedProblems    int64                          `json:"attempted_problems"`
	AcceptedSubmissions  int64                          `json:"accepted_submissions"`
	SolvedProblems       int64                          `json:"solved_problems"`
	AcceptanceRate       float64                        `json:"acceptance_rate"`
	VerdictDistribution  []ProfileStatsVerdictResponse  `json:"verdict_distribution"`
	LanguageDistribution []ProfileStatsLanguageResponse `json:"language_distribution"`
	Activity             []ProfileStatsActivityResponse `json:"activity"`
}

type ProfileStatsVerdictResponse struct {
	Verdict string `json:"verdict"`
	Count   int64  `json:"count"`
}

type ProfileStatsLanguageResponse struct {
	Language string `json:"language"`
	Count    int64  `json:"count"`
}

type ProfileStatsActivityResponse struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
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
