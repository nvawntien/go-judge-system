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
			"language",
			"status",
			"created_at",
		},
		values: values,
	}
}

func TestSubmissionRepositoryListByUserAppliesOwnerFiltersAndPagination(t *testing.T) {
	createdAt := time.Date(2026, 7, 25, 13, 0, 0, 0, time.UTC)
	problemID := int64(42)
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
			[]driver.Value{int64(123), problemID, "Two Sum", "GO", "PENDING", createdAt},
			[]driver.Value{int64(122), problemID, "Two Sum", "GO", "PENDING", createdAt},
		), nil
	})

	got, err := (&submissionRepository{db: db}).ListByUser(
		ctx,
		outbound.ListSubmissionsFilter{
			UserID:    "actor",
			Status:    "PENDING",
			Language:  "GO",
			ProblemID: &problemID,
			Limit:     2,
			Offset:    1,
		},
	)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if got.Total != 3 || len(got.Items) != 2 {
		t.Fatalf("result = %+v, want total 3 and 2 items", got)
	}
	if got.Items[0].ID != 123 ||
		got.Items[0].ProblemName != "Two Sum" ||
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
			t.Fatalf("query %d does not contain mandatory actor owner value: %+v", index, queryArgs[index])
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
}

func TestSubmissionRepositoryListByUserReturnsNonNilEmptyItems(t *testing.T) {
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

	got, err := (&submissionRepository{db: db}).ListByUser(
		context.Background(),
		outbound.ListSubmissionsFilter{UserID: "actor", Limit: 20},
	)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if got.Total != 0 || got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("result = %#v, want zero total and non-nil empty items", got)
	}
}

func TestSubmissionRepositoryListByUserRequiresOwner(t *testing.T) {
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		t.Fatal("database must not be queried without an owner")
		return nil, nil
	})

	_, err := (&submissionRepository{db: db}).ListByUser(
		context.Background(),
		outbound.ListSubmissionsFilter{Limit: 20},
	)
	if err == nil || !strings.Contains(err.Error(), "user ID is required") {
		t.Fatalf("ListByUser() error = %v, want required owner error", err)
	}
}

func TestSubmissionRepositoryListByUserWrapsDatabaseErrors(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name          string
		failQueryCall int
		wantOperation string
	}{
		{name: "count", failQueryCall: 1, wantOperation: "count submissions by user"},
		{name: "items", failQueryCall: 2, wantOperation: "list submissions by user"},
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

			_, err := (&submissionRepository{db: db}).ListByUser(
				context.Background(),
				outbound.ListSubmissionsFilter{UserID: "actor", Limit: 20},
			)
			if !errors.Is(err, databaseErr) || !strings.Contains(err.Error(), tt.wantOperation) {
				t.Fatalf("ListByUser() error = %v, want wrapped %q", err, tt.wantOperation)
			}
		})
	}
}

func TestSubmissionRepositoryListByUserPreservesCancellation(t *testing.T) {
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

	_, err := (&submissionRepository{db: db}).ListByUser(
		ctx,
		outbound.ListSubmissionsFilter{UserID: "actor", Limit: 20},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ListByUser() error = %v, want context canceled", err)
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
