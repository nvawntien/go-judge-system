package usecase

import (
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/domain/entity"
)

// Shared mapper functions to avoid duplication across use cases.

func MapProblemToResponse(p *entity.Problem, includePrivate bool) dto.ProblemResponse {
	resp := dto.ProblemResponse{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Description: p.Description,
		Difficulty:  string(p.Difficulty),
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

func MapTestCaseToResponse(tc *entity.TestCase) dto.TestCaseResponse {
	return dto.TestCaseResponse{
		ID:             tc.ID,
		ProblemID:      tc.ProblemID,
		Input:          tc.Input,
		ExpectedOutput: tc.ExpectedOutput,
		IsExample:      tc.IsExample,
		Order:          tc.Order,
	}
}
