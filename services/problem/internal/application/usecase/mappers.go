package usecase

import (
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/domain/entity"
)

func MapProblemToResponse(p *entity.Problem, includePrivate bool) dto.ProblemResponse {
	examples := make([]dto.ProblemExampleDTO, 0, len(p.Examples))
	for _, ex := range p.Examples {
		examples = append(examples, dto.ProblemExampleDTO{
			Input:       ex.Input,
			Output:      ex.Output,
			Explanation: ex.Explanation,
		})
	}

	resp := dto.ProblemResponse{
		ID:          p.ID,
		Slug:        p.TitleSlug,
		Title:       p.Title,
		Description: p.Description,
		Difficulty:  string(p.Difficulty),
		Examples:    examples,
		Constraints: p.Constraints,
		Hints:       p.Hints,
		TimeLimit:   p.TimeLimit,
		MemoryLimit: p.MemoryLimit,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if includePrivate {
		resp.AuthorID = p.AuthorID
		resp.IsHidden = p.IsHidden
	}
	return resp
}

func MapProblemToAdminDetailResponse(p *entity.Problem, tc *entity.TestCase) dto.AdminProblemDetailResponse {
	examples := make([]dto.ProblemExampleDTO, 0, len(p.Examples))
	for _, ex := range p.Examples {
		examples = append(examples, dto.ProblemExampleDTO{
			Input:       ex.Input,
			Output:      ex.Output,
			Explanation: ex.Explanation,
		})
	}

	var deletedAt *string
	if p.DeletedAt != nil {
		value := p.DeletedAt.Format("2006-01-02T15:04:05Z")
		deletedAt = &value
	}

	resp := dto.AdminProblemDetailResponse{
		ID:          p.ID,
		Slug:        p.TitleSlug,
		Title:       p.Title,
		Description: p.Description,
		Difficulty:  string(p.Difficulty),
		Examples:    examples,
		Constraints: p.Constraints,
		Hints:       p.Hints,
		TimeLimit:   p.TimeLimit,
		MemoryLimit: p.MemoryLimit,
		AuthorID:    p.AuthorID,
		IsHidden:    p.IsHidden,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		DeletedAt:   deletedAt,
		TestCase: dto.AdminTestCaseMetadataResponse{
			HasTestCase: false,
		},
	}

	if tc != nil {
		testCaseCreatedAt := tc.CreatedAt.Format("2006-01-02T15:04:05Z")
		testCaseUpdatedAt := tc.UpdatedAt.Format("2006-01-02T15:04:05Z")

		resp.TestCase = dto.AdminTestCaseMetadataResponse{
			HasTestCase:  true,
			ID:           int64Ptr(tc.ID),
			ProblemID:    int64Ptr(tc.ProblemID),
			ZipObjectKey: stringPtr(tc.ZipObjectKey),
			TestCount:    intPtr(tc.TestCount),
			Version:      intPtr(tc.Version),
			CreatedAt:    stringPtr(testCaseCreatedAt),
			UpdatedAt:    stringPtr(testCaseUpdatedAt),
		}
	}

	return resp
}

func MapExampleDTOsToEntity(dtos []dto.ProblemExampleDTO) []entity.ProblemExample {
	examples := make([]entity.ProblemExample, 0, len(dtos))
	for _, d := range dtos {
		examples = append(examples, entity.ProblemExample{
			Input:       d.Input,
			Output:      d.Output,
			Explanation: d.Explanation,
		})
	}
	return examples
}

func int64Ptr(value int64) *int64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
