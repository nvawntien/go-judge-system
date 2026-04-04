package dto

type ProblemSlugRequest struct {
	Slug string `uri:"slug" binding:"required"`
}

type ProblemIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type ProblemExampleDTO struct {
	Input       string `json:"input" binding:"required"`
	Output      string `json:"output" binding:"required"`
	Explanation string `json:"explanation,omitempty"`
}

type CreateProblemRequest struct {
	Title       string              `json:"title" binding:"required,min=3"`
	Slug        string              `json:"slug" binding:"required,min=3"`
	Description string              `json:"description" binding:"required,min=3"`
	Difficulty  string              `json:"difficulty" binding:"required,oneof=EASY MEDIUM HARD"`
	Examples    []ProblemExampleDTO `json:"examples" binding:"required,min=1,dive"`
	Constraints string              `json:"constraints"`
	Hints       []string            `json:"hints"`
	TimeLimit   float64             `json:"time_limit" binding:"required,gt=0,max=30"`
	MemoryLimit int                 `json:"memory_limit" binding:"required,min=16,max=1024"`
}

type CreateProblemResponse struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
}

type UpdateProblemRequest struct {
	Title       *string              `json:"title,omitempty" binding:"omitempty,min=3"`
	NewSlug     *string              `json:"slug,omitempty" binding:"omitempty,min=3"`
	Description *string              `json:"description,omitempty" binding:"omitempty,min=3"`
	Difficulty  *string              `json:"difficulty,omitempty" binding:"omitempty,oneof=EASY MEDIUM HARD"`
	Examples    *[]ProblemExampleDTO `json:"examples,omitempty" binding:"omitempty,min=1,dive"`
	Constraints *string              `json:"constraints,omitempty"`
	Hints       *[]string            `json:"hints,omitempty"`
	TimeLimit   *float64             `json:"time_limit,omitempty" binding:"omitempty,gt=0,max=30"`
	MemoryLimit *int                 `json:"memory_limit,omitempty" binding:"omitempty,min=16,max=1024"`
}

type ProblemResponse struct {
	ID          int64               `json:"id"`
	Slug        string              `json:"slug"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Difficulty  string              `json:"difficulty"`
	Examples    []ProblemExampleDTO `json:"examples,omitempty"`
	Constraints string              `json:"constraints,omitempty"`
	Hints       []string            `json:"hints,omitempty"`
	TimeLimit   float64             `json:"time_limit"`
	MemoryLimit int                 `json:"memory_limit"`
	AuthorID    string              `json:"author_id,omitempty"`
	IsHidden    bool                `json:"is_hidden,omitempty"`
	CreatedAt   string              `json:"created_at"`
}

type ProblemDetailResponse struct {
	ProblemResponse
}

type ListProblemsRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	Limit      int    `form:"limit,default=20" binding:"min=1,max=100"`
	Difficulty string `form:"difficulty" binding:"omitempty,oneof=EASY MEDIUM HARD"`
	Search     string `form:"search"`
}

type ListProblemsResponse struct {
	Items []ProblemResponse `json:"items"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}
