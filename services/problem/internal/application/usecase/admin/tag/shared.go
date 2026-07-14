package tag

import (
	"context"
	"regexp"
	"strings"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

var nonSlugCharsPattern = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeTagName(value string) string {
	return strings.TrimSpace(value)
}

func normalizeTagDescription(value string) string {
	return strings.TrimSpace(value)
}

func normalizeTagSlug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlugCharsPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "tag"
	}
	return slug
}

func ensureTagCanBeDeactivated(ctx context.Context, tagRepo outbound.TagRepository, tag *entity.Tag) error {
	if !tag.IsActive {
		return nil
	}

	count, err := tagRepo.CountPublishedProblemsByTagID(ctx, tag.ID)
	if err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}
	if count > 0 {
		return domain.ErrTagUsedByPublishedProblem
	}

	return nil
}
