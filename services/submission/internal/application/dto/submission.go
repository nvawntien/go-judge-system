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
	SourceCode  string `json:"source_code"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type ListMySubmissionsResponse struct {
	Items []SubmissionResponse `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}
