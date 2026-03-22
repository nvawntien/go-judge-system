package judge

// ResultMessage represents a judge result published from Worker back to Submission Service.
// This typed struct replaces the previous untyped map[string]interface{} to ensure
// compile-time safety and consistent contract between services.
type ResultMessage struct {
	SubmissionID  int64                `json:"submission_id"`
	AttemptID     string               `json:"attempt_id"`
	Status        string               `json:"status"`
	CompileOutput *string              `json:"compile_output,omitempty"`
	ExecutionTime *int                 `json:"execution_time,omitempty"`
	MemoryUsed    *int                 `json:"memory_used,omitempty"`
	Error         *string              `json:"error,omitempty"`
	TestCases     []TestCaseResultItem `json:"test_cases"`
}

// TestCaseResultItem represents an individual testcase execution result.
type TestCaseResultItem struct {
	TestCaseID    int64   `json:"test_case_id"`
	Status        string  `json:"status"`
	ActualOutput  *string `json:"actual_output,omitempty"`
	ExecutionTime *int    `json:"execution_time,omitempty"`
	MemoryUsed    *int    `json:"memory_used,omitempty"`
	Order         int     `json:"order"`
}
