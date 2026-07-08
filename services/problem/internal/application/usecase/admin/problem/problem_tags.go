package problem

import (
	"context"
	"fmt"
	"sort"

	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

const maxProblemTags = 20

func resolveProblemTags(ctx context.Context, tagRepo outbound.TagRepository, rawTagIDs []uint) ([]entity.Tag, error) {
	tagIDs, err := normalizeTagIDs(rawTagIDs)
	if err != nil {
		return nil, err
	}
	if len(tagIDs) == 0 {
		return nil, nil
	}

	tags, err := tagRepo.ListByIDs(ctx, tagIDs, true)
	if err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	byID := make(map[uint]*entity.Tag, len(tags))
	for _, tag := range tags {
		byID[tag.ID] = tag
	}

	missing := make([]uint, 0)
	resolved := make([]entity.Tag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		tag, ok := byID[tagID]
		if !ok {
			missing = append(missing, tagID)
			continue
		}
		resolved = append(resolved, *tag)
	}

	if len(missing) > 0 {
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		return nil, response.NewAppError(
			response.CodeBadRequest,
			fmt.Sprintf("tag_ids contain invalid or inactive ids: %v", missing),
			nil,
		)
	}

	return resolved, nil
}

func normalizeTagIDs(rawTagIDs []uint) ([]uint, error) {
	if len(rawTagIDs) == 0 {
		return []uint{}, nil
	}

	seen := make(map[uint]struct{}, len(rawTagIDs))
	tagIDs := make([]uint, 0, len(rawTagIDs))
	for _, tagID := range rawTagIDs {
		if tagID == 0 {
			return nil, response.NewAppError(response.CodeBadRequest, "tag_ids must contain positive integers", nil)
		}
		if _, ok := seen[tagID]; ok {
			continue
		}
		seen[tagID] = struct{}{}
		tagIDs = append(tagIDs, tagID)
		if len(tagIDs) > maxProblemTags {
			return nil, response.NewAppError(
				response.CodeBadRequest,
				fmt.Sprintf("problem supports at most %d tags", maxProblemTags),
				nil,
			)
		}
	}

	return tagIDs, nil
}

func hasInactiveTags(tags []entity.Tag) bool {
	for _, tag := range tags {
		if !tag.IsActive {
			return true
		}
	}
	return false
}
