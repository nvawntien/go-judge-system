package dto

type JudgeResultMessage struct {
	SubmissionID  int64                 `json:"submission_id"`
	Status        string                `json:"status"`
	CompileOutput *string               `json:"compile_output,omitempty"`
	ExecutionTime *int                  `json:"execution_time,omitempty"`
	MemoryUsed    *int                  `json:"memory_used,omitempty"`
	Results       []JudgeTestCaseResult `json:"results"`
}

type JudgeTestCaseResult struct {
	TestCaseID    int64   `json:"test_case_id"`
	Status        string  `json:"status"`
	ActualOutput  *string `json:"actual_output,omitempty"`
	ExecutionTime *int    `json:"execution_time,omitempty"`
	MemoryUsed    *int    `json:"memory_used,omitempty"`
	Order         int     `json:"order"`
}
