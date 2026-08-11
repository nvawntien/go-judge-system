package tags

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type Repository interface {
	Create(context.Context, *entity.Tag) error
	GetBySlug(context.Context, string) (*entity.Tag, error)
}

type Result struct {
	Created        int
	AlreadyExisted int
	Failed         int
}

func (r Result) Total() int {
	return r.Created + r.AlreadyExisted + r.Failed
}

// Seed creates missing tags by slug without altering pre-existing rows.
func Seed(ctx context.Context, repo Repository, definitions []Definition, output io.Writer) (Result, error) {
	var result Result
	for _, definition := range definitions {
		existing, err := repo.GetBySlug(ctx, definition.Slug)
		switch {
		case err == nil:
			result.AlreadyExisted++
			if existing.Name != definition.Name {
				fmt.Fprintf(output, "WARNING: slug %q already exists as %q; preserving it instead of desired name %q\n", definition.Slug, existing.Name, definition.Name)
			}
			continue
		case !errors.Is(err, domain.ErrTagNotFound):
			result.Failed++
			fmt.Fprintf(output, "ERROR: inspect tag %q: %v\n", definition.Slug, err)
			continue
		}

		if err := repo.Create(ctx, entity.NewTag(definition.Name, definition.Slug, definition.Description, definition.Active)); err != nil {
			if errors.Is(err, domain.ErrTagAlreadyExists) {
				result.AlreadyExisted++
				continue
			}
			result.Failed++
			fmt.Fprintf(output, "ERROR: create tag %q: %v\n", definition.Slug, err)
			continue
		}
		result.Created++
	}

	if result.Failed > 0 {
		return result, fmt.Errorf("failed to seed %d tag(s)", result.Failed)
	}
	return result, nil
}
