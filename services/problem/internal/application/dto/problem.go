package dto

import "encoding/json"

type ProblemSlugRequest struct {
	Slug string `uri:"slug" binding:"required"`
}

type ProblemIDRequest struct {
	ID int64 `uri:"problem_id" binding:"required,min=1"`
}

type ProblemExampleDTO struct {
	Input          string `json:"input" binding:"required"`
	ExpectedOutput string `json:"expected_output" binding:"required"`
	Explanation    string `json:"explanation,omitempty"`
}

func (d *ProblemExampleDTO) UnmarshalJSON(data []byte) error {
	type problemExampleAlias ProblemExampleDTO
	var raw struct {
		problemExampleAlias
		LegacyOutput string `json:"output"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*d = ProblemExampleDTO(raw.problemExampleAlias)
	if d.ExpectedOutput == "" {
		d.ExpectedOutput = raw.LegacyOutput
	}
	return nil
}

type CreateProblemRequest struct {
	Title       string              `json:"title" binding:"required,min=3"`
	Description string              `json:"description" binding:"required,min=3"`
	Difficulty  string              `json:"difficulty" binding:"required"`
	TagIDs      []uint              `json:"tag_ids" binding:"omitempty,max=20,dive,min=1"`
	Examples    []ProblemExampleDTO `json:"examples" binding:"required,min=1,dive"`
	Constraints []string            `json:"constraints"`
	Hints       []string            `json:"hints"`
	TimeLimit   float64             `json:"time_limit"`
	MemoryLimit int                 `json:"memory_limit"`
}

type CreateProblemResponse struct {
	ProblemResponse
}

type UpdateProblemRequest struct {
	Title       *string              `json:"title,omitempty" binding:"omitempty,min=3"`
	NewSlug     *string              `json:"slug,omitempty" binding:"omitempty,min=3"`
	Description *string              `json:"description,omitempty" binding:"omitempty,min=3"`
	Difficulty  *string              `json:"difficulty,omitempty" binding:"omitempty,oneof=easy medium hard"`
	TagIDs      *[]uint              `json:"tag_ids,omitempty" binding:"omitempty,max=20,dive,min=1"`
	Examples    *[]ProblemExampleDTO `json:"examples,omitempty" binding:"omitempty,min=1,dive"`
	Constraints *[]string            `json:"constraints,omitempty"`
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
	Tags        []TagResponse       `json:"tags,omitempty"`
	Examples    []ProblemExampleDTO `json:"examples,omitempty"`
	Constraints []string            `json:"constraints,omitempty"`
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

type AdminProblemDetailResponse struct {
	ID          int64               `json:"id"`
	Slug        string              `json:"slug"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Difficulty  string              `json:"difficulty"`
	Tags        []TagResponse       `json:"tags"`
	Examples    []ProblemExampleDTO `json:"examples"`
	Constraints []string            `json:"constraints"`
	Hints       []string            `json:"hints"`

	TimeLimit   float64 `json:"time_limit"`
	MemoryLimit int     `json:"memory_limit"`

	AuthorID string `json:"author_id"`
	IsHidden bool   `json:"is_hidden"`

	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DeletedAt *string `json:"deleted_at,omitempty"`

	TestCase AdminTestCaseMetadataResponse `json:"testcase"`
}

type AdminTestCaseMetadataResponse struct {
	HasTestCase  bool    `json:"has_testcase"`
	ID           *int64  `json:"id,omitempty"`
	ProblemID    *int64  `json:"problem_id,omitempty"`
	ZipObjectKey *string `json:"zip_object_key,omitempty"`
	TestCount    *int    `json:"test_count,omitempty"`
	Version      *int    `json:"version,omitempty"`
	CreatedAt    *string `json:"created_at,omitempty"`
	UpdatedAt    *string `json:"updated_at,omitempty"`
}

type ListProblemsRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	Limit      int    `form:"limit,default=20" binding:"min=1,max=100"`
	Difficulty string `form:"difficulty" binding:"omitempty,oneof=easy medium hard"`
	Search     string `form:"search"`
	TagSlug    string `form:"tag_slug" json:"tag_slug" binding:"omitempty"`
}

type ListProblemsResponse struct {
	Items []ProblemResponse `json:"items"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}
