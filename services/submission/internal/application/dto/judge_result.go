package dto

type JudgeResultMessage struct {
	SubmissionID  int64                 `json:"submission_id"`
	Status        string                `json:"status"`
	CompileOutput *string               `json:"compile_output,omitempty"`
	ExecutionTime *int                  `json:"execution_time,omitempty"`
	MemoryUsed    *int                  `json:"memory_used,omitempty"`
	Results       []JudgeTestCaseResult `json:"test_cases"`
}

type JudgeTestCaseResult struct {
	Index          int     `json:"index"`
	Status         string  `json:"status"`
	ActualOutput   *string `json:"actual_output,omitempty"`
	Input          *string `json:"input,omitempty"`
	ExpectedOutput *string `json:"expected_output,omitempty"`
	ExecutionTime  *int    `json:"execution_time,omitempty"`
	MemoryUsed     *int    `json:"memory_used,omitempty"`
}
