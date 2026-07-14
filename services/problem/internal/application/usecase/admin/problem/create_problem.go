package problem

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type createProblemUseCase struct {
	problemRepo outbound.ProblemRepository
	tagRepo     outbound.TagRepository
}

const (
	defaultTimeLimit   = 1000
	defaultMemoryLimit = 256
)

var nonSlugCharsPattern = regexp.MustCompile(`[^a-z0-9]+`)

func NewCreateProblemUseCase(problemRepo outbound.ProblemRepository, tagRepo outbound.TagRepository) inbound.CreateProblemUseCase {
	return &createProblemUseCase{
		problemRepo: problemRepo,
		tagRepo:     tagRepo,
	}
}

func (uc *createProblemUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.CreateProblemRequest) (dto.ProblemDetailResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleContributor) {
		return dto.ProblemDetailResponse{}, domain.ErrForbidden
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return dto.ProblemDetailResponse{}, response.NewAppError(response.CodeBadRequest, "title is required", nil)
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		return dto.ProblemDetailResponse{}, response.NewAppError(response.CodeBadRequest, "description is required", nil)
	}

	difficulty, err := normalizeDifficulty(req.Difficulty)
	if err != nil {
		return dto.ProblemDetailResponse{}, err
	}

	tags, err := resolveProblemTags(ctx, uc.tagRepo, req.TagIDs)
	if err != nil {
		return dto.ProblemDetailResponse{}, err
	}

	slug, err := uc.generateUniqueSlug(ctx, title)
	if err != nil {
		return dto.ProblemDetailResponse{}, err
	}

	timeLimit := req.TimeLimit
	if timeLimit <= 0 {
		timeLimit = defaultTimeLimit
	}

	memoryLimit := req.MemoryLimit
	if memoryLimit <= 0 {
		memoryLimit = defaultMemoryLimit
	}

	examples := usecase.MapExampleDTOsToEntity(req.Examples)

	problem := entity.NewProblem(
		title,
		slug,
		description,
		difficulty,
		tags,
		examples, req.Constraints, req.Hints,
		timeLimit, memoryLimit,
		claims.UserID,
	)

	problem.IsHidden = true

	if err := uc.problemRepo.Create(ctx, problem); err != nil {
		if errors.Is(err, domain.ErrProblemAlreadyExists) {
			return dto.ProblemDetailResponse{}, domain.ErrProblemAlreadyExists
		}

		return dto.ProblemDetailResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return dto.ProblemDetailResponse{
		ProblemResponse: usecase.MapProblemToResponse(problem, true),
	}, nil
}

func normalizeDifficulty(raw string) (entity.Difficulty, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(entity.Easy):
		return entity.Easy, nil
	case string(entity.Medium):
		return entity.Medium, nil
	case string(entity.Hard):
		return entity.Hard, nil
	default:
		return "", response.NewAppError(response.CodeBadRequest, "difficulty must be one of easy, medium, hard", nil)
	}
}

func (uc *createProblemUseCase) generateUniqueSlug(ctx context.Context, title string) (string, error) {
	base := slugify(title)
	for attempt := 1; ; attempt++ {
		candidate := base
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", base, attempt)
		}

		_, err := uc.problemRepo.GetBySlug(ctx, candidate)
		if err == nil {
			continue
		}
		if errors.Is(err, domain.ErrProblemNotFound) {
			return candidate, nil
		}
		return "", domain.ErrInternalServer.Wrap(err)
	}
}

func slugify(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = nonSlugCharsPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "problem"
	}
	return slug
}
