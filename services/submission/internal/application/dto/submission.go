package dto

type CreateSubmissionRequest struct {
	ProblemID  int64  `json:"problem_id" binding:"required,min=1"`
	Language   string `json:"language" binding:"required,oneof=C CPP JAVA PYTHON GO JAVASCRIPT"`
	SourceCode string `json:"source_code" binding:"required,min=1"`
}

type CreateSubmissionResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}
