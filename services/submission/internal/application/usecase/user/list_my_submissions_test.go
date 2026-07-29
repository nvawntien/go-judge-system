package user

import (
	"context"
	"encoding/json"
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

type fakeListMySubmissionsRepository struct {
	*fakeSubmissionRepository
	result       outbound.ListSubmissionsResult
	err          error
	summaryErr   error
	summaries    map[int64]outbound.SubmissionResultSummary
	filter       outbound.ListSubmissionsFilter
	calls        int
	summaryCalls int
	summaryIDs   []int64
}

func newFakeListMySubmissionsRepository() *fakeListMySubmissionsRepository {
	return &fakeListMySubmissionsRepository{
		fakeSubmissionRepository: &fakeSubmissionRepository{},
	}
}

func (r *fakeListMySubmissionsRepository) List(
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

func (r *fakeListMySubmissionsRepository) ResultSummaries(
	ctx context.Context,
	submissionIDs []int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	r.summaryCalls++
	r.summaryIDs = append([]int64(nil), submissionIDs...)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.summaryErr != nil {
		return nil, r.summaryErr
	}
	if r.summaries == nil {
		return map[int64]outbound.SubmissionResultSummary{}, nil
	}
	return r.summaries, nil
}

func intPointer(value int) *int          { return &value }
func int64Pointer(value int64) *int64    { return &value }
func stringPointer(value string) *string { return &value }

func listSubmissionFixture() *entity.Submission {
	executionTime := 12
	memoryUsed := 4300
	return &entity.Submission{
		ID:            123,
		ProblemID:     42,
		ProblemName:   "Two Sum",
		UserID:        "actor",
		Username:      "actor-name",
		Language:      entity.LanguageGo,
		SourceCode:    "must not be exposed",
		Status:        entity.StatusAccepted,
		ExecutionTime: &executionTime,
		MemoryUsed:    &memoryUsed,
		CreatedAt:     time.Date(2026, 7, 25, 13, 0, 0, 0, time.UTC),
	}
}

func TestListMySubmissionsSupportedRolesRemainOwnerScoped(t *testing.T) {
	for _, role := range []rbac.Role{
		rbac.RoleUser,
		rbac.RoleContributor,
		rbac.RoleModerator,
		rbac.RoleAdmin,
	} {
		t.Run(string(role), func(t *testing.T) {
			repo := newFakeListMySubmissionsRepository()
			repo.result = outbound.ListSubmissionsResult{
				Items: []*entity.Submission{listSubmissionFixture()},
				Total: 1,
			}
			repo.summaries = map[int64]outbound.SubmissionResultSummary{
				123: {SubmissionID: 123, Passed: 20, Total: 20},
			}

			got, err := NewListMySubmissionsUseCase(repo).Execute(
				context.Background(),
				auth.Claims{UserID: "actor", Role: role},
				dto.ListMySubmissionsRequest{},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if repo.calls != 1 {
				t.Fatalf("repository calls = %d, want 1", repo.calls)
			}
			if repo.summaryCalls != 1 || len(repo.summaryIDs) != 1 || repo.summaryIDs[0] != 123 {
				t.Fatalf("summary calls/ids = %d/%v, want 1/[123]", repo.summaryCalls, repo.summaryIDs)
			}
			if repo.filter.UserID == nil || *repo.filter.UserID != "actor" {
				t.Fatalf("repository UserID = %v, want authenticated actor", repo.filter.UserID)
			}
			if repo.filter.Offset != 0 || repo.filter.Limit != defaultSubmissionsLimit {
				t.Fatalf("repository pagination = %+v, want default page/limit", repo.filter)
			}
			if len(got.Items) != 1 {
				t.Fatalf("items length = %d, want 1", len(got.Items))
			}
			if got.Items[0].ProblemTitle != "Two Sum" {
				t.Fatalf("ProblemTitle = %q, want ProblemName snapshot", got.Items[0].ProblemTitle)
			}
			payload, err := json.Marshal(got.Items[0])
			if err != nil {
				t.Fatalf("marshal list item: %v", err)
			}
			if string(payload) == "" || json.Valid(payload) == false {
				t.Fatalf("marshal list item produced invalid json: %s", payload)
			}
			if containsJSONField(payload, "source_code") {
				t.Fatalf("list item leaked source_code field: %s", payload)
			}
			if got.Items[0].ExecutionTimeMS == nil || *got.Items[0].ExecutionTimeMS != 12 ||
				got.Items[0].MemoryUsedKB == nil || *got.Items[0].MemoryUsedKB != 4300 ||
				got.Items[0].PassedTestCases == nil || *got.Items[0].PassedTestCases != 20 ||
				got.Items[0].TotalTestCases == nil || *got.Items[0].TotalTestCases != 20 {
				t.Fatalf("metadata = %+v, want runtime/memory/pass counts", got.Items[0])
			}
			if got.Pagination != (dto.PaginationResponse{
				Page:       1,
				Limit:      20,
				Total:      1,
				TotalPages: 1,
			}) {
				t.Fatalf("pagination = %+v", got.Pagination)
			}
		})
	}
}

func containsJSONField(payload []byte, field string) bool {
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return false
	}
	_, ok := decoded[field]
	return ok
}

func TestListMySubmissionsFiltersAndPagination(t *testing.T) {
	tests := []struct {
		name     string
		claims   auth.Claims
		req      dto.ListMySubmissionsRequest
		want     outbound.ListSubmissionsFilter
		total    int64
		wantPage dto.PaginationResponse
	}{
		{
			name:   "status filter",
			claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			req:    dto.ListMySubmissionsRequest{Status: "ACCEPTED"},
			want: outbound.ListSubmissionsFilter{
				UserID: stringPointer("actor"), Status: stringPointer("ACCEPTED"), Limit: 20,
			},
			total:    1,
			wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name:   "language filter",
			claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			req:    dto.ListMySubmissionsRequest{Language: "GO"},
			want: outbound.ListSubmissionsFilter{
				UserID: stringPointer("actor"), Language: stringPointer("GO"), Limit: 20,
			},
			total:    1,
			wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name:   "problem filter",
			claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			req:    dto.ListMySubmissionsRequest{ProblemID: int64Pointer(42)},
			want: outbound.ListSubmissionsFilter{
				UserID: stringPointer("actor"), ProblemID: int64Pointer(42), Limit: 20,
			},
			total:    1,
			wantPage: dto.PaginationResponse{Page: 1, Limit: 20, Total: 1, TotalPages: 1},
		},
		{
			name:   "combined filters and custom pagination",
			claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser},
			req: dto.ListMySubmissionsRequest{
				Page: intPointer(2), Limit: intPointer(3),
				Status: "PENDING", Language: "CPP", ProblemID: int64Pointer(42),
			},
			want: outbound.ListSubmissionsFilter{
				UserID: stringPointer("actor"), Status: stringPointer("PENDING"), Language: stringPointer("CPP"),
				ProblemID: int64Pointer(42), Limit: 3, Offset: 3,
			},
			total:    7,
			wantPage: dto.PaginationResponse{Page: 2, Limit: 3, Total: 7, TotalPages: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeListMySubmissionsRepository()
			repo.result = outbound.ListSubmissionsResult{
				Items: []*entity.Submission{},
				Total: tt.total,
			}

			got, err := NewListMySubmissionsUseCase(repo).Execute(context.Background(), tt.claims, tt.req)
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

func TestListMySubmissionsRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		claims  auth.Claims
		req     dto.ListMySubmissionsRequest
		wantErr error
	}{
		{name: "zero page", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Page: intPointer(0)}, wantErr: domain.ErrInvalidPage},
		{name: "negative page", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Page: intPointer(-1)}, wantErr: domain.ErrInvalidPage},
		{name: "zero limit", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Limit: intPointer(0)}, wantErr: domain.ErrInvalidLimit},
		{name: "negative limit", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Limit: intPointer(-1)}, wantErr: domain.ErrInvalidLimit},
		{name: "limit over maximum", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Limit: intPointer(101)}, wantErr: domain.ErrInvalidLimit},
		{name: "zero problem ID", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{ProblemID: int64Pointer(0)}, wantErr: domain.ErrInvalidProblemID},
		{name: "negative problem ID", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{ProblemID: int64Pointer(-1)}, wantErr: domain.ErrInvalidProblemID},
		{name: "invalid status", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Status: "DONE"}, wantErr: domain.ErrInvalidSubmissionStatus},
		{name: "invalid language", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Language: "RUST"}, wantErr: domain.ErrInvalidLanguage},
		{name: "non-executable language", claims: auth.Claims{UserID: "actor", Role: rbac.RoleUser}, req: dto.ListMySubmissionsRequest{Language: "C"}, wantErr: domain.ErrInvalidLanguage},
		{
			name:    "missing actor user ID",
			claims:  auth.Claims{Role: rbac.RoleUser},
			req:     dto.ListMySubmissionsRequest{},
			wantErr: domain.ErrSubmissionUnauthenticated,
		},
		{
			name:    "missing actor role",
			claims:  auth.Claims{UserID: "actor"},
			req:     dto.ListMySubmissionsRequest{},
			wantErr: domain.ErrSubmissionUnauthenticated,
		},
		{
			name:    "unsupported role",
			claims:  auth.Claims{UserID: "actor", Role: rbac.Role("auditor")},
			req:     dto.ListMySubmissionsRequest{},
			wantErr: domain.ErrSubmissionForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeListMySubmissionsRepository()
			_, err := NewListMySubmissionsUseCase(repo).Execute(context.Background(), tt.claims, tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if repo.calls != 0 {
				t.Fatalf("repository calls = %d, want 0", repo.calls)
			}
		})
	}
}

func TestListMySubmissionsRepositoryFailureIsMappedSafely(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	repo := newFakeListMySubmissionsRepository()
	repo.err = databaseErr

	_, err := NewListMySubmissionsUseCase(repo).Execute(
		context.Background(),
		auth.Claims{UserID: "actor", Role: rbac.RoleUser},
		dto.ListMySubmissionsRequest{},
	)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, databaseErr) {
		t.Fatalf("Execute() error = %v, want safely wrapped repository error", err)
	}
}

func TestListMySubmissionsSummaryFailureIsMappedSafely(t *testing.T) {
	databaseErr := errors.New("summary unavailable")
	repo := newFakeListMySubmissionsRepository()
	repo.result = outbound.ListSubmissionsResult{
		Items: []*entity.Submission{listSubmissionFixture()},
		Total: 1,
	}
	repo.summaryErr = databaseErr

	_, err := NewListMySubmissionsUseCase(repo).Execute(
		context.Background(),
		auth.Claims{UserID: "actor", Role: rbac.RoleUser},
		dto.ListMySubmissionsRequest{},
	)
	if !errors.Is(err, domain.ErrInternalServer) || !errors.Is(err, databaseErr) {
		t.Fatalf("Execute() error = %v, want safely wrapped summary error", err)
	}
}

func TestListMySubmissionsPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newFakeListMySubmissionsRepository()
	_, err := NewListMySubmissionsUseCase(repo).Execute(
		ctx,
		auth.Claims{UserID: "actor", Role: rbac.RoleUser},
		dto.ListMySubmissionsRequest{},
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
	if got.Limit != want.Limit ||
		got.Offset != want.Offset {
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
