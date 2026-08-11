package tags

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"testing"

	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

var kebabCaseSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type memoryRepository struct {
	tags map[string]*entity.Tag
}

func newMemoryRepository(tags ...*entity.Tag) *memoryRepository {
	repo := &memoryRepository{tags: make(map[string]*entity.Tag, len(tags))}
	for _, item := range tags {
		repo.tags[item.Slug] = item
	}
	return repo
}

func (r *memoryRepository) GetBySlug(_ context.Context, slug string) (*entity.Tag, error) {
	tag, ok := r.tags[slug]
	if !ok {
		return nil, domain.ErrTagNotFound
	}
	return tag, nil
}

func (r *memoryRepository) Create(_ context.Context, tag *entity.Tag) error {
	if _, exists := r.tags[tag.Slug]; exists {
		return domain.ErrTagAlreadyExists
	}
	r.tags[tag.Slug] = tag
	return nil
}

func TestDesiredDefinitionsAreCanonicalAndValid(t *testing.T) {
	definitions := Desired()
	if len(definitions) == 0 {
		t.Fatal("Desired() returned no tags")
	}

	seen := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if definition.Name == "" || definition.Description == "" {
			t.Fatalf("tag %q has an empty name or description", definition.Slug)
		}
		if !definition.Active {
			t.Fatalf("tag %q is not active", definition.Slug)
		}
		if !kebabCaseSlug.MatchString(definition.Slug) {
			t.Fatalf("tag %q is not valid kebab-case", definition.Slug)
		}
		if _, exists := seen[definition.Slug]; exists {
			t.Fatalf("duplicate slug %q", definition.Slug)
		}
		seen[definition.Slug] = struct{}{}
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	definitions := Desired()
	repo := newMemoryRepository()

	first, err := Seed(context.Background(), repo, definitions, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("first seed error: %v", err)
	}
	if first.Created != len(definitions) || first.AlreadyExisted != 0 || first.Failed != 0 {
		t.Fatalf("first result = %+v, want all tags created", first)
	}

	second, err := Seed(context.Background(), repo, definitions, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("second seed error: %v", err)
	}
	if second.Created != 0 || second.AlreadyExisted != len(definitions) || second.Failed != 0 {
		t.Fatalf("second result = %+v, want all tags preserved", second)
	}
}

func TestSeedPreservesExistingConflictingTag(t *testing.T) {
	repo := newMemoryRepository(entity.NewTag("DP", "dynamic-programming", "Existing custom description.", false))
	definitions := []Definition{{
		Name:        "Dynamic Programming",
		Slug:        "dynamic-programming",
		Description: "Problems decomposed into overlapping subproblems with stored optimal results.",
		Active:      true,
	}}
	var output bytes.Buffer

	result, err := Seed(context.Background(), repo, definitions, &output)
	if err != nil {
		t.Fatalf("seed error: %v", err)
	}
	if result.Created != 0 || result.AlreadyExisted != 1 || result.Failed != 0 {
		t.Fatalf("result = %+v", result)
	}
	existing, err := repo.GetBySlug(context.Background(), "dynamic-programming")
	if err != nil || existing.Name != "DP" || existing.IsActive {
		t.Fatalf("existing tag was changed: %+v, err=%v", existing, err)
	}
	if !bytes.Contains(output.Bytes(), []byte("WARNING:")) {
		t.Fatalf("expected conflict warning, got %q", output.String())
	}
}

func TestSeedCountsCreationFailures(t *testing.T) {
	repo := &failingRepository{}
	result, err := Seed(context.Background(), repo, []Definition{tag("Implementation", "implementation", "A description.")}, &bytes.Buffer{})
	if err == nil || result.Failed != 1 || result.Total() != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

type failingRepository struct{}

func (failingRepository) GetBySlug(context.Context, string) (*entity.Tag, error) {
	return nil, domain.ErrTagNotFound
}

func (failingRepository) Create(context.Context, *entity.Tag) error {
	return errors.New("database unavailable")
}
