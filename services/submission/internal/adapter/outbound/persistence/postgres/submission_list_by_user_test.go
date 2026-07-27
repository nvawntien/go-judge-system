package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"
)

func countRows(total int64) driver.Rows {
	return &repositoryReadRows{
		columns: []string{"count"},
		values:  [][]driver.Value{{total}},
	}
}

func submissionListRows(values ...[]driver.Value) driver.Rows {
	return &repositoryReadRows{
		columns: []string{
			"id",
			"problem_id",
			"problem_name",
			"user_id",
			"username",
			"language",
			"status",
			"created_at",
		},
		values: values,
	}
}

func TestSubmissionRepositoryListAppliesFiltersAndPagination(t *testing.T) {
	createdAt := time.Date(2026, 7, 25, 13, 0, 0, 0, time.UTC)
	problemID := int64(42)
	userID := "actor"
	status := "PENDING"
	language := "GO"
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")

	var queries []string
	var queryArgs [][]driver.NamedValue
	var contextValues []any
	db := newRepositoryReadGormDB(t, func(
		queryCtx context.Context,
		query string,
		args []driver.NamedValue,
	) (driver.Rows, error) {
		queries = append(queries, query)
		queryArgs = append(queryArgs, args)
		contextValues = append(contextValues, queryCtx.Value(contextKey{}))
		if len(queries) == 1 {
			return countRows(3), nil
		}
		return submissionListRows(
			[]driver.Value{int64(123), problemID, "Two Sum", userID, "actor-name", language, status, createdAt},
			[]driver.Value{int64(122), problemID, "Two Sum", userID, "actor-name", language, status, createdAt},
		), nil
	})

	got, err := (&submissionRepository{db: db}).List(
		ctx,
		outbound.ListSubmissionsFilter{
			UserID:    &userID,
			Status:    &status,
			Language:  &language,
			ProblemID: &problemID,
			Limit:     2,
			Offset:    1,
		},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got.Total != 3 || len(got.Items) != 2 {
		t.Fatalf("result = %+v, want total 3 and 2 items", got)
	}
	if got.Items[0].ID != 123 ||
		got.Items[0].ProblemName != "Two Sum" ||
		got.Items[0].UserID != userID ||
		got.Items[0].Username != "actor-name" ||
		got.Items[0].Language != entity.LanguageGo ||
		got.Items[0].Status != entity.StatusPending {
		t.Fatalf("first item = %+v, want compact DAO mapping", got.Items[0])
	}
	if len(queries) != 2 {
		t.Fatalf("query calls = %d, want count + item query", len(queries))
	}
	for index, query := range queries {
		normalized := strings.ToUpper(query)
		for _, predicate := range []string{"USER_ID =", "STATUS =", "LANGUAGE =", "PROBLEM_ID ="} {
			if !strings.Contains(normalized, predicate) {
				t.Fatalf("query %d missing %q predicate: %s", index, predicate, query)
			}
		}
		if contextValues[index] != "request-context" {
			t.Fatalf("query %d context value = %v", index, contextValues[index])
		}
		if !containsDriverValue(queryArgs[index], "actor") {
			t.Fatalf("query %d does not contain user ID filter value: %+v", index, queryArgs[index])
		}
	}

	countQuery := strings.ToUpper(queries[0])
	if strings.Contains(countQuery, "LIMIT") || strings.Contains(countQuery, "OFFSET") {
		t.Fatalf("count query contains pagination: %s", queries[0])
	}

	itemQuery := strings.ToUpper(queries[1])
	if !strings.Contains(itemQuery, "ORDER BY CREATED_AT DESC, ID DESC") {
		t.Fatalf("item query lacks deterministic ordering: %s", queries[1])
	}
	if !strings.Contains(itemQuery, "LIMIT") || !strings.Contains(itemQuery, "OFFSET") {
		t.Fatalf("item query lacks limit/offset: %s", queries[1])
	}
	if strings.Contains(itemQuery, "SOURCE_CODE") {
		t.Fatalf("item query selects source_code: %s", queries[1])
	}
	if !strings.Contains(itemQuery, "USER_ID") || !strings.Contains(itemQuery, "USERNAME") {
		t.Fatalf("item query must select user identity snapshots: %s", queries[1])
	}
}

func TestSubmissionRepositoryListWithoutFiltersReturnsAllRows(t *testing.T) {
	createdAt := time.Date(2026, 7, 25, 13, 0, 0, 0, time.UTC)
	var queries []string
	db := newRepositoryReadGormDB(t, func(
		_ context.Context,
		query string,
		_ []driver.NamedValue,
	) (driver.Rows, error) {
		queries = append(queries, query)
		if len(queries) == 1 {
			return countRows(2), nil
		}
		return submissionListRows(
			[]driver.Value{int64(123), int64(42), "Two Sum", "user-1", "alice", "GO", "PENDING", createdAt},
			[]driver.Value{int64(122), int64(43), "Three Sum", "user-2", "bob", "CPP", "ACCEPTED", createdAt},
		), nil
	})

	got, err := (&submissionRepository{db: db}).List(
		context.Background(),
		outbound.ListSubmissionsFilter{Limit: 20},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("result = %+v, want all rows", got)
	}
	for _, query := range queries {
		normalized := strings.ToUpper(query)
		for _, predicate := range []string{"USER_ID =", "STATUS =", "LANGUAGE =", "PROBLEM_ID ="} {
			if strings.Contains(normalized, predicate) {
				t.Fatalf("unfiltered query contains %q predicate: %s", predicate, query)
			}
		}
	}
}

func TestSubmissionRepositoryListAppliesIndividualFilters(t *testing.T) {
	userID := "user-123"
	status := "ACCEPTED"
	language := "GO"
	problemID := int64(42)

	tests := []struct {
		name          string
		filter        outbound.ListSubmissionsFilter
		wantPredicate string
		wantValue     driver.Value
	}{
		{name: "user ID", filter: outbound.ListSubmissionsFilter{UserID: &userID, Limit: 20}, wantPredicate: "USER_ID =", wantValue: userID},
		{name: "status", filter: outbound.ListSubmissionsFilter{Status: &status, Limit: 20}, wantPredicate: "STATUS =", wantValue: status},
		{name: "language", filter: outbound.ListSubmissionsFilter{Language: &language, Limit: 20}, wantPredicate: "LANGUAGE =", wantValue: language},
		{name: "problem ID", filter: outbound.ListSubmissionsFilter{ProblemID: &problemID, Limit: 20}, wantPredicate: "PROBLEM_ID =", wantValue: problemID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queries []string
			var queryArgs [][]driver.NamedValue
			db := newRepositoryReadGormDB(t, func(
				_ context.Context,
				query string,
				args []driver.NamedValue,
			) (driver.Rows, error) {
				queries = append(queries, query)
				queryArgs = append(queryArgs, args)
				if len(queries) == 1 {
					return countRows(0), nil
				}
				return submissionListRows(), nil
			})

			_, err := (&submissionRepository{db: db}).List(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(queries) != 2 {
				t.Fatalf("query calls = %d, want count + item query", len(queries))
			}
			for index, query := range queries {
				if !strings.Contains(strings.ToUpper(query), tt.wantPredicate) {
					t.Fatalf("query %d missing %q: %s", index, tt.wantPredicate, query)
				}
				if !containsDriverValue(queryArgs[index], tt.wantValue) {
					t.Fatalf("query %d args = %+v, want %v", index, queryArgs[index], tt.wantValue)
				}
			}
		})
	}
}

func TestSubmissionRepositoryListReturnsNonNilEmptyItems(t *testing.T) {
	queryCalls := 0
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		queryCalls++
		if queryCalls == 1 {
			return countRows(0), nil
		}
		return submissionListRows(), nil
	})

	got, err := (&submissionRepository{db: db}).List(
		context.Background(),
		outbound.ListSubmissionsFilter{Limit: 20},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got.Total != 0 || got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("result = %#v, want zero total and non-nil empty items", got)
	}
}

func TestSubmissionRepositoryListWrapsDatabaseErrors(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name          string
		failQueryCall int
		wantOperation string
	}{
		{name: "count", failQueryCall: 1, wantOperation: "count submissions"},
		{name: "items", failQueryCall: 2, wantOperation: "list submissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryCalls := 0
			db := newRepositoryReadGormDB(t, func(
				context.Context,
				string,
				[]driver.NamedValue,
			) (driver.Rows, error) {
				queryCalls++
				if queryCalls == tt.failQueryCall {
					return nil, databaseErr
				}
				return countRows(1), nil
			})

			_, err := (&submissionRepository{db: db}).List(
				context.Background(),
				outbound.ListSubmissionsFilter{Limit: 20},
			)
			if !errors.Is(err, databaseErr) || !strings.Contains(err.Error(), tt.wantOperation) {
				t.Fatalf("List() error = %v, want wrapped %q", err, tt.wantOperation)
			}
		})
	}
}

func TestSubmissionRepositoryListPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		t.Fatal("database query must not run after cancellation")
		return nil, nil
	})

	_, err := (&submissionRepository{db: db}).List(
		ctx,
		outbound.ListSubmissionsFilter{Limit: 20},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List() error = %v, want context canceled", err)
	}
}

func containsDriverValue(values []driver.NamedValue, want driver.Value) bool {
	for _, value := range values {
		if value.Value == want {
			return true
		}
	}
	return false
}
