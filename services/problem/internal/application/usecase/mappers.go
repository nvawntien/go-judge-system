package usecase

import (
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/domain/entity"
)

func MapProblemToResponse(p *entity.Problem, includePrivate bool) dto.ProblemResponse {
	examples := make([]dto.ProblemExampleDTO, 0, len(p.Examples))
	for _, ex := range p.Examples {
		examples = append(examples, dto.ProblemExampleDTO{
			Input:          ex.Input,
			ExpectedOutput: ex.Output,
			Explanation:    ex.Explanation,
		})
	}

	tags := make([]dto.TagResponse, 0, len(p.Tags))
	for _, tag := range p.Tags {
		tags = append(tags, MapTagToResponse(&tag))
	}

	resp := dto.ProblemResponse{
		ID:          p.ID,
		Slug:        p.TitleSlug,
		Title:       p.Title,
		Description: p.Description,
		Difficulty:  string(p.Difficulty),
		Tags:        tags,
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
			Input:          ex.Input,
			ExpectedOutput: ex.Output,
			Explanation:    ex.Explanation,
		})
	}

	tags := make([]dto.TagResponse, 0, len(p.Tags))
	for _, tag := range p.Tags {
		tags = append(tags, MapTagToResponse(&tag))
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
		Tags:        tags,
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

func MapTagToResponse(tag *entity.Tag) dto.TagResponse {
	return dto.TagResponse{
		ID:          tag.ID,
		Name:        tag.Name,
		Slug:        tag.Slug,
		Description: tag.Description,
	}
}

func MapTagToAdminResponse(tag *entity.Tag) dto.AdminTagResponse {
	return dto.AdminTagResponse{
		TagResponse: MapTagToResponse(tag),
		IsActive:    tag.IsActive,
		CreatedAt:   tag.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   tag.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func MapExampleDTOsToEntity(dtos []dto.ProblemExampleDTO) []entity.ProblemExample {
	examples := make([]entity.ProblemExample, 0, len(dtos))
	for _, d := range dtos {
		examples = append(examples, entity.ProblemExample{
			Input:       d.Input,
			Output:      d.ExpectedOutput,
			Explanation: d.Explanation,
		})
	}
	return examples
}

func MapTestCaseToMetadataResponse(tc *entity.TestCase) dto.TestCaseMetadataResponse {
	return dto.TestCaseMetadataResponse{
		ID:           tc.ID,
		ProblemID:    tc.ProblemID,
		ZipObjectKey: tc.ZipObjectKey,
		TestCount:    tc.TestCount,
		Version:      tc.Version,
		CreatedAt:    tc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    tc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
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
