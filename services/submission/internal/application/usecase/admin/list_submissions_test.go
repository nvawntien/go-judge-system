package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

type fakeListAdminSubmissionsRepository struct {
	result outbound.ListSubmissionsResult
	err    error
	filter outbound.ListSubmissionsFilter
	calls  int
}

func (r *fakeListAdminSubmissionsRepository) Create(context.Context, *entity.Submission) error {
	return nil
}

func (r *fakeListAdminSubmissionsRepository) GetByID(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}

func (r *fakeListAdminSubmissionsRepository) GetByIDForUpdate(context.Context, int64) (*entity.Submission, error) {
	return nil, nil
}

func (r *fakeListAdminSubmissionsRepository) Update(context.Context, *entity.Submission) error {
	return nil
}

func (r *fakeListAdminSubmissionsRepository) List(
	ctx context.Context,
	filter outbound.ListSubmissionsFilter,
) (outbound.ListSubmissionsResult, error) {
	r.calls++
	r.filter = filter
	if err := ctx.Err(); err != nil {
		return outbound.ListSubmissionsResult{}, err
	}
	return r.result, r.err
}

func (r *fakeListAdminSubmissionsRepository) ResultSummaries(
	context.Context,
	[]int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	return map[int64]outbound.SubmissionResultSummary{}, nil
}

func intPointer(value int) *int          { return &value }
func int64Pointer(value int64) *int64    { return &value }
func stringPointer(value string) *string { return &value }

func adminSubmissionFixture() *entity.Submission {
	return &entity.Submission{
		ID:          123,
		ProblemID:   42,
		ProblemName: "Two Sum",
		UserID:      "user-123",
		Username:    "vantien",
		Language:    entity.LanguageGo,
		SourceCode:  "must not be exposed",
		Status:      entity.StatusPending,
		CreatedAt:   time.Date(2026, 7, 25, 14, 0, 0, 0, time.UTC),
	}
}

func TestListAdminSubmissionsAllowedRolesListSystemWide(t *testing.T) {
	for _, role := range []rbac.Role{rbac.RoleModerator, rbac.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			repo := &fakeListAdminSubmissionsRepository{
				result: outbound.ListSubmissionsResult{
					Items: []*entity.Submission{adminSubmissionFixture()},
					Total: 1,
				},
			}

			got, err := NewListAdminSubmissionsUseCase(repo).Execute(
				context.Background(),
				auth.Claims{UserID: "actor", Role: role},
				dto.ListAdminSubmissionsRequest{},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if repo.calls != 1 {
				t.Fatalf("repository calls = %d, want 1", repo.calls)
			}
			if repo.filter.UserID != nil {
				t.Fatalf("system-wide list must not use actor owner filter: %+v", repo.filter)
			}
			if repo.filter.Offset != 0 || repo.filter.Limit != defaultSubmissionsLimit {
				t.Fatalf("pagination filter = %+v, want defaults", repo.filter)
			}
			if len(got.Items) != 1 {
				t.Fatalf("items length = %d, want 1", len(got.Items))
			}
			item := got.Items[0]
			if item.UserID != "user-123" || item.Username != "vantien" || item.ProblemTitle != "Two Sum" {
				t.Fatalf("item mapping = %+v, want stored snapshots", item)
			}
			if got.Pagination != (dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1}) {
				t.Fatalf("pagination = %+v", got.Pagination)
			}
		})
	}
}

func TestListAdminSubmissionsRejectsUnauthorizedAndForbiddenRoles(t *testing.T) {
	tests := []struct {
		name    string
		claims  auth.Claims
		wantErr error
	}{
		{name: "missing actor user ID", claims: auth.Claims{Role: rbac.RoleModerator}, wantErr: domain.ErrSubmissionUnauthenticated},
		{name: "missing actor role", claims: auth.Claims{UserID: "actor"}, wantErr: domain.ErrSubmissionUnauthenticated},
		{name: "user forbidden", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, wantErr: domain.ErrSubmissionForbidden},
		{name: "contributor forbidden", claims: auth.Claims{UserID: "actor", Role: rbac.RoleContributor}, wantErr: domain.ErrSubmissionForbidden},
		{name: "unsupported role forbidden", claims: auth.Claims{UserID: "actor", Role: rbac.Role("auditor")}, wantErr: domain.ErrSubmissionForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeListAdminSubmissionsRepository{}
			_, err := NewListAdminSubmissionsUseCase(repo).Execute(
				context.Background(),
				tt.claims,
				dto.ListAdminSubmissionsRequest{},
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if repo.calls != 0 {
				t.Fatalf("repository calls = %d, want 0", repo.calls)
			}
		})
	}
}

func TestListAdminSubmissionsFiltersAndPagination(t *testing.T) {
	tests := []struct {
		name     string
		req      dto.ListAdminSubmissionsRequest
		want     outbound.ListSubmissionsFilter
		total    int64
		wantPage dto.PaginationResponse
	}{
		{
			name:  "valid custom pagination",
			req:   dto.ListAdminSubmissionsRequest{Page: intPointer(2), Limit: intPointer(3)},
			want:  outbound.ListSubmissionsFilter{Limit: 3, Offset: 3},
			total: 7, wantPage: dto.PaginationResponse{Page: 2, Limit: 3, Total: 7, TotalPages: 3},
		},
		{
			name:  "valid status filter",
			req:   dto.ListAdminSubmissionsRequest{Status: stringPointer("PENDING")},
			want:  outbound.ListSubmissionsFilter{Status: stringPointer("PENDING"), Limit: 20},
			total: 1, wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name:  "valid language filter",
			req:   dto.ListAdminSubmissionsRequest{Language: stringPointer("C")},
			want:  outbound.ListSubmissionsFilter{Language: stringPointer("C"), Limit: 20},
			total: 1, wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name:  "valid problem ID filter",
			req:   dto.ListAdminSubmissionsRequest{ProblemID: int64Pointer(42)},
			want:  outbound.ListSubmissionsFilter{ProblemID: int64Pointer(42), Limit: 20},
			total: 1, wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name:  "valid user ID filter",
			req:   dto.ListAdminSubmissionsRequest{UserID: stringPointer(" user-123 ")},
			want:  outbound.ListSubmissionsFilter{UserID: stringPointer("user-123"), Limit: 20},
			total: 1, wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name: "combined filters",
			req: dto.ListAdminSubmissionsRequest{
				Page: intPointer(2), Limit: intPointer(3),
				Status: stringPointer("ACCEPTED"), Language: stringPointer("GO"),
				ProblemID: int64Pointer(42), UserID: stringPointer("user-123"),
			},
			want: outbound.ListSubmissionsFilter{
				UserID: stringPointer("user-123"), Status: stringPointer("ACCEPTED"),
				Language: stringPointer("GO"), ProblemID: int64Pointer(42),
				Limit: 3, Offset: 3,
			},
			total: 7, wantPage: dto.PaginationResponse{Page: 2, Limit: 3, Total: 7, TotalPages: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeListAdminSubmissionsRepository{
				result: outbound.ListSubmissionsResult{Items: []*entity.Submission{}, Total: tt.total},
			}

			got, err := NewListAdminSubmissionsUseCase(repo).Execute(
				context.Background(),
				auth.Claims{UserID: "moderator", Role: rbac.RoleModerator},
				tt.req,
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			assertListFilter(t, repo.filter, tt.want)
			if got.Pagination != tt.wantPage {
				t.Fatalf("pagination = %+v, want %+v", got.Pagination, tt.wantPage)
			}
			if got.Items == nil || len(got.Items) != 0 {
				t.Fatalf("items = %#v, want non-nil empty slice", got.Items)
			}
		})
	}
}

func TestListAdminSubmissionsRejectsInvalidQueryValues(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.ListAdminSubmissionsRequest
		wantErr error
	}{
		{name: "page zero", req: dto.ListAdminSubmissionsRequest{Page: intPointer(0)}, wantErr: domain.ErrInvalidPage},
		{name: "negative page", req: dto.ListAdminSubmissionsRequest{Page: intPointer(-1)}, wantErr: domain.ErrInvalidPage},
		{name: "limit zero", req: dto.ListAdminSubmissionsRequest{Limit: intPointer(0)}, wantErr: domain.ErrInvalidLimit},
		{name: "negative limit", req: dto.ListAdminSubmissionsRequest{Limit: intPointer(-1)}, wantErr: domain.ErrInvalidLimit},
		{name: "limit above maximum", req: dto.ListAdminSubmissionsRequest{Limit: intPointer(101)}, wantErr: domain.ErrInvalidLimit},
		{name: "empty status", req: dto.ListAdminSubmissionsRequest{Status: stringPointer("")}, wantErr: domain.ErrInvalidSubmissionStatus},
		{name: "invalid status", req: dto.ListAdminSubmissionsRequest{Status: stringPointer("DONE")}, wantErr: domain.ErrInvalidSubmissionStatus},
		{name: "empty language", req: dto.ListAdminSubmissionsRequest{Language: stringPointer(" ")}, wantErr: domain.ErrInvalidLanguage},
		{name: "invalid language", req: dto.ListAdminSubmissionsRequest{Language: stringPointer("RUST")}, wantErr: domain.ErrInvalidLanguage},
		{name: "zero problem ID", req: dto.ListAdminSubmissionsRequest{ProblemID: int64Pointer(0)}, wantErr: domain.ErrInvalidProblemID},
		{name: "negative problem ID", req: dto.ListAdminSubmissionsRequest{ProblemID: int64Pointer(-1)}, wantErr: domain.ErrInvalidProblemID},
		{name: "blank user ID", req: dto.ListAdminSubmissionsRequest{UserID: stringPointer(" ")}, wantErr: domain.ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeListAdminSubmissionsRepository{}
			_, err := NewListAdminSubmissionsUseCase(repo).Execute(
				context.Background(),
				auth.Claims{UserID: "moderator", Role: rbac.RoleModerator},
				tt.req,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if repo.calls != 0 {
				t.Fatalf("repository calls = %d, want 0", repo.calls)
			}
		})
	}
}

func TestListAdminSubmissionsRepositoryFailureIsMappedSafely(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	repo := &fakeListAdminSubmissionsRepository{err: databaseErr}

	_, err := NewListAdminSubmissionsUseCase(repo).Execute(
		context.Background(),
		auth.Claims{UserID: "admin", Role: rbac.RoleAdmin},
		dto.ListAdminSubmissionsRequest{},
	)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, databaseErr) {
		t.Fatalf("Execute() error = %v, want safely wrapped repository error", err)
	}
}

func TestListAdminSubmissionsPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := &fakeListAdminSubmissionsRepository{}
	_, err := NewListAdminSubmissionsUseCase(repo).Execute(
		ctx,
		auth.Claims{UserID: "admin", Role: rbac.RoleAdmin},
		dto.ListAdminSubmissionsRequest{},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context canceled", err)
	}
	if errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("context cancellation must not be remapped: %v", err)
	}
}

func assertListFilter(
	t *testing.T,
	got outbound.ListSubmissionsFilter,
	want outbound.ListSubmissionsFilter,
) {
	t.Helper()
	if got.Limit != want.Limit || got.Offset != want.Offset {
		t.Fatalf("filter = %+v, want %+v", got, want)
	}
	assertStringPointer(t, "UserID", got.UserID, want.UserID)
	assertStringPointer(t, "Status", got.Status, want.Status)
	assertStringPointer(t, "Language", got.Language, want.Language)
	if (got.ProblemID == nil) != (want.ProblemID == nil) {
		t.Fatalf("ProblemID = %v, want %v", got.ProblemID, want.ProblemID)
	}
	if got.ProblemID != nil && *got.ProblemID != *want.ProblemID {
		t.Fatalf("ProblemID = %d, want %d", *got.ProblemID, *want.ProblemID)
	}
}

func assertStringPointer(t *testing.T, name string, got *string, want *string) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	if got != nil && *got != *want {
		t.Fatalf("%s = %q, want %q", name, *got, *want)
	}
}
