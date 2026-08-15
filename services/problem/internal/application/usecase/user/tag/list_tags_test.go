package tag

import (
	"context"
	"testing"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain/entity"
)

type activeTagCatalogRepository struct {
	outbound.TagRepository
	listActiveCalls int
}

func (r *activeTagCatalogRepository) ListActive(context.Context) ([]*entity.Tag, error) {
	r.listActiveCalls++
	return []*entity.Tag{
		{ID: 1, Name: "Graphs", Slug: "graphs", IsActive: true},
		{ID: 2, Name: "Dijkstra", Slug: "dijkstra", IsActive: true},
	}, nil
}

func TestListTagsReturnsOnlyThePublicAuthoringCatalog(t *testing.T) {
	t.Parallel()
	repository := &activeTagCatalogRepository{}
	response, err := NewListTagsUseCase(repository).Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if repository.listActiveCalls != 1 {
		t.Fatalf("ListActive calls = %d, want 1", repository.listActiveCalls)
	}
	if len(response.Items) != 2 {
		t.Fatalf("items = %+v, want 2 active tags", response.Items)
	}
	if response.Items[0].ID != 1 || response.Items[0].Name != "Graphs" || response.Items[0].Slug != "graphs" {
		t.Fatalf("first tag = %+v, want public tag fields", response.Items[0])
	}
}
