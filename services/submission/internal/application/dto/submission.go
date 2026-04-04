package dto

type CreateSubmissionRequest struct {
	ProblemID   int64  `json:"problem_id" binding:"required,min=1"`
	ProblemName string `json:"problem_name" binding:"required,min=1"`
	Language    string `json:"language" binding:"required,oneof=C CPP JAVA PYTHON GO JAVASCRIPT"`
	SourceCode  string `json:"source_code" binding:"required,min=1"`
}

type ListMySubmissionsRequest struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	Limit    int    `form:"limit,default=20" binding:"min=1,max=100"`
	Status   string `form:"status" binding:"omitempty"`
	Language string `form:"language" binding:"omitempty"`
}

type SubmissionResponse struct {
	ID          int64  `json:"id"`
	ProblemID   int64  `json:"problem_id"`
	ProblemName string `json:"problem_name"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	Language    string `json:"language"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type ListMySubmissionsResponse struct {
	Items []SubmissionResponse `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

type SubmissionIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type ProblemIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type ListProblemSubmissionsQueryRequest struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	Limit    int    `form:"limit,default=20" binding:"min=1,max=100"`
	Status   string `form:"status" binding:"omitempty"`
	Language string `form:"language" binding:"omitempty"`
}

type ListProblemSubmissionsResponse struct {
	Items []SubmissionResponse `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

type ListSubmissionsRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	Limit     int    `form:"limit,default=20" binding:"min=1,max=100"`
	ProblemID *int64 `form:"problem_id" binding:"omitempty,min=1"`
	UserID    string `form:"user_id" binding:"omitempty"`
	Status    string `form:"status" binding:"omitempty"`
	Language  string `form:"language" binding:"omitempty"`
}

type ListSubmissionsResponse struct {
	Items []SubmissionResponse `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

type SubmissionResultResponse struct {
	ID            int64   `json:"id"`
	TestIndex     int     `json:"test_index"`
	Status        string  `json:"status"`
	ActualOutput  *string `json:"actual_output,omitempty"`
	ExecutionTime *int    `json:"execution_time,omitempty"`
	MemoryUsed    *int    `json:"memory_used,omitempty"`
}

type SubmissionDetailResponse struct {
	SubmissionResponse
	SourceCode    string                     `json:"source_code"`
	ExecutionTime *int                       `json:"execution_time,omitempty"`
	MemoryUsed    *int                       `json:"memory_used,omitempty"`
	CompileOutput *string                    `json:"compile_output,omitempty"`
	Results       []SubmissionResultResponse `json:"results"`
}
