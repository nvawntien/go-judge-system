package dto

type TestCaseIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type CreateTestCaseRequest struct {
	Input          string `json:"input" binding:"required"`
	ExpectedOutput string `json:"expected_output" binding:"required"`
	IsExample      bool   `json:"is_example"`
	Order          int    `json:"order" binding:"required,min=1"`
}

type CreateTestCaseResponse struct {
	ID int64 `json:"id"`
}

type UpdateTestCaseRequest struct {
	Input          *string `json:"input,omitempty"`
	ExpectedOutput *string `json:"expected_output,omitempty"`
	IsExample      *bool   `json:"is_example,omitempty"`
	Order          *int    `json:"order,omitempty" binding:"omitempty,min=1"`
}

type TestCaseResponse struct {
	ID             int64  `json:"id"`
	ProblemID      int64  `json:"problem_id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	IsExample      bool   `json:"is_example"`
	Order          int    `json:"order"`
}

type TestCaseListResponse struct {
	Items []TestCaseResponse `json:"items"`
	Total int                `json:"total"`
}
