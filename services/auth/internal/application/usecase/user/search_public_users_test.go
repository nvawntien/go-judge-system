package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestSearchPublicUsersUsesBoundedPublicFilterAndMapsOnlyVisibleItems(t *testing.T) {
	avatar := "https://avatar.example/ada.png"
	repo := &publicSearchRepository{
		result: outbound.SearchPublicUsersResult{
			Items: []*entity.User{
				{Username: "ada", FullName: "Ada Lovelace", AvatarURL: &avatar, Rating: 1800, IsActive: true},
				{Username: "hidden", FullName: "Not Verified", IsActive: false},
				{Username: "suspended", FullName: "Suspended", IsActive: true, IsSuspended: true},
			},
			Total: 1,
		},
	}

	result, err := NewSearchPublicUsersUseCase(repo).Execute(context.Background(), dto.SearchPublicUsersRequest{Query: "  Ada  "})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.filter.Query != "Ada" || repo.filter.Limit != 10 || repo.filter.Offset != 0 {
		t.Fatalf("repository filter = %+v", repo.filter)
	}
	if result.Pagination.Page != 1 || result.Pagination.Limit != 10 || result.Pagination.Total != 1 || result.Pagination.TotalPages != 1 {
		t.Fatalf("pagination = %+v", result.Pagination)
	}
	if len(result.Items) != 1 || result.Items[0] != (dto.PublicUserSearchItem{Username: "ada", FullName: "Ada Lovelace", AvatarURL: &avatar, Rating: 1800}) {
		t.Fatalf("items = %+v", result.Items)
	}
}

func TestSearchPublicUsersSupportsPaginationNoMatchBlankAndLiteralWildcards(t *testing.T) {
	page, limit := 2, 3
	repo := &publicSearchRepository{result: outbound.SearchPublicUsersResult{
		Items: []*entity.User{
			{Username: "alpha", FullName: "Alpha User", IsActive: true},
			{Username: "beta", FullName: "Beta User", IsActive: true},
		},
		Total: 7,
	}}
	uc := NewSearchPublicUsersUseCase(repo)

	result, err := uc.Execute(context.Background(), dto.SearchPublicUsersRequest{Query: `%_\\`, Page: &page, Limit: &limit})
	if err != nil {
		t.Fatalf("wildcard query error = %v", err)
	}
	if repo.filter.Query != `%_\\` || repo.filter.Offset != 3 || repo.filter.Limit != 3 || result.Pagination.TotalPages != 3 {
		t.Fatalf("wildcard/pagination result=%+v filter=%+v", result, repo.filter)
	}
	if got := []string{result.Items[0].Username, result.Items[1].Username}; strings.Join(got, ",") != "alpha,beta" {
		t.Fatalf("result ordering changed: %v", got)
	}

	repo.called = false
	blank, err := uc.Execute(context.Background(), dto.SearchPublicUsersRequest{Query: " \t "})
	if err != nil || repo.called || blank.Items == nil || len(blank.Items) != 0 || blank.Pagination.Total != 0 {
		t.Fatalf("blank query result=%+v err=%v called=%t", blank, err, repo.called)
	}

	repo.result = outbound.SearchPublicUsersResult{}
	noMatch, err := uc.Execute(context.Background(), dto.SearchPublicUsersRequest{Query: "nobody"})
	if err != nil || noMatch.Items == nil || len(noMatch.Items) != 0 || noMatch.Pagination.TotalPages != 0 {
		t.Fatalf("no-match result=%+v err=%v", noMatch, err)
	}
}

func TestSearchPublicUsersRejectsInvalidBoundsAndMapsRepositoryErrors(t *testing.T) {
	uc := NewSearchPublicUsersUseCase(&publicSearchRepository{})
	zero, overLimit := 0, 21
	tooLong := strings.Repeat("a", maxPublicUserSearchQueryRunes+1)
	for _, req := range []dto.SearchPublicUsersRequest{{Query: "ada", Page: &zero}, {Query: "ada", Limit: &overLimit}, {Query: tooLong}} {
		if _, err := uc.Execute(context.Background(), req); err == nil {
			t.Fatalf("Execute(%+v) error = nil", req)
		}
	}

	_, err := NewSearchPublicUsersUseCase(&publicSearchRepository{err: errors.New("database unavailable")}).Execute(
		context.Background(), dto.SearchPublicUsersRequest{Query: "ada"},
	)
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("repository error = %v, want internal", err)
	}
}

type publicSearchRepository struct {
	profileHardeningRepository
	result outbound.SearchPublicUsersResult
	err    error
	filter outbound.SearchPublicUsersFilter
	called bool
}

func (r *publicSearchRepository) SearchPublicUsers(_ context.Context, filter outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	r.called = true
	r.filter = filter
	return r.result, r.err
}

var _ outbound.UserRepository = (*publicSearchRepository)(nil)
