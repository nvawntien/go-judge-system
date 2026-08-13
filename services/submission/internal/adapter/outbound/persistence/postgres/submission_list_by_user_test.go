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
			"execution_time",
			"memory_used",
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
			[]driver.Value{int64(123), problemID, "Two Sum", userID, "actor-name", language, status, int64(12), int64(4300), createdAt},
			[]driver.Value{int64(122), problemID, "Two Sum", userID, "actor-name", language, status, int64(8), int64(4100), createdAt},
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
		got.Items[0].Status != entity.StatusPending ||
		got.Items[0].ExecutionTime == nil || *got.Items[0].ExecutionTime != 12 ||
		got.Items[0].MemoryUsed == nil || *got.Items[0].MemoryUsed != 4300 {
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
	if !strings.Contains(itemQuery, "EXECUTION_TIME") || !strings.Contains(itemQuery, "MEMORY_USED") {
		t.Fatalf("item query must select execution metadata: %s", queries[1])
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
			[]driver.Value{int64(123), int64(42), "Two Sum", "user-1", "alice", "GO", "PENDING", nil, nil, createdAt},
			[]driver.Value{int64(122), int64(43), "Three Sum", "user-2", "bob", "CPP", "ACCEPTED", int64(15), int64(4096), createdAt},
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

func TestSubmissionRepositoryResultSummariesAggregatesCounts(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")

	var query string
	var queryArgs []driver.NamedValue
	var contextValue any
	db := newRepositoryReadGormDB(t, func(
		queryCtx context.Context,
		actualQuery string,
		args []driver.NamedValue,
	) (driver.Rows, error) {
		query = actualQuery
		queryArgs = args
		contextValue = queryCtx.Value(contextKey{})
		return &repositoryReadRows{
			columns: []string{"submission_id", "passed", "total"},
			values: [][]driver.Value{
				{int64(123), int64(20), int64(20)},
				{int64(122), int64(4), int64(20)},
			},
		}, nil
	})

	got, err := (&submissionRepository{db: db}).ResultSummaries(ctx, []int64{123, 122})
	if err != nil {
		t.Fatalf("ResultSummaries() error = %v", err)
	}
	if contextValue != "request-context" {
		t.Fatalf("query context value = %v, want request-context", contextValue)
	}
	if got[123] != (outbound.SubmissionResultSummary{SubmissionID: 123, Passed: 20, Total: 20}) ||
		got[122] != (outbound.SubmissionResultSummary{SubmissionID: 122, Passed: 4, Total: 20}) {
		t.Fatalf("summaries = %+v", got)
	}

	normalized := strings.ToUpper(query)
	for _, fragment := range []string{
		"SUBMISSION_ID",
		"SUM(CASE WHEN SUBMISSION_RESULTS.STATUS =",
		"COUNT(*) AS TOTAL",
		"JOIN SUBMISSIONS",
		"ATTEMPT_ID = SUBMISSIONS.CURRENT_ATTEMPT_ID",
		"GROUP BY",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Fatalf("summary query missing %q: %s", fragment, query)
		}
	}
	for _, sensitive := range []string{"INPUT", "EXPECTED", "ACTUAL_OUTPUT", "STDOUT", "STDERR"} {
		if strings.Contains(normalized, sensitive) {
			t.Fatalf("summary query selects hidden/sensitive testcase data: %s", query)
		}
	}
	if !containsDriverValue(queryArgs, string(entity.ResultAccepted)) ||
		!containsDriverValue(queryArgs, int64(123)) ||
		!containsDriverValue(queryArgs, int64(122)) {
		t.Fatalf("summary query args = %+v, want accepted status and submission IDs", queryArgs)
	}
}

func TestSubmissionRepositoryResultSummariesUsesOnlyCurrentAttempt(t *testing.T) {
	var query string
	db := newRepositoryReadGormDB(t, func(
		_ context.Context,
		actualQuery string,
		_ []driver.NamedValue,
	) (driver.Rows, error) {
		query = actualQuery
		// The current attempt has four accepted results. Historical attempt rows
		// are intentionally absent from this grouped result.
		return &repositoryReadRows{
			columns: []string{"submission_id", "passed", "total"},
			values:  [][]driver.Value{{int64(77), int64(4), int64(4)}},
		}, nil
	})

	got, err := (&submissionRepository{db: db}).ResultSummaries(context.Background(), []int64{77})
	if err != nil {
		t.Fatalf("ResultSummaries() error = %v", err)
	}
	if got[77] != (outbound.SubmissionResultSummary{SubmissionID: 77, Passed: 4, Total: 4}) {
		t.Fatalf("summary = %+v, want current attempt 4/4", got[77])
	}

	normalized := strings.ToUpper(query)
	if !strings.Contains(normalized, "JOIN SUBMISSIONS") ||
		!strings.Contains(normalized, "SUBMISSION_RESULTS.ATTEMPT_ID = SUBMISSIONS.CURRENT_ATTEMPT_ID") {
		t.Fatalf("summary query is not scoped to the current attempt: %s", query)
	}
}

func TestSubmissionRepositoryResultSummariesSkipsEmptyIDs(t *testing.T) {
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		t.Fatal("database query must not run for empty ID slice")
		return nil, nil
	})

	got, err := (&submissionRepository{db: db}).ResultSummaries(context.Background(), nil)
	if err != nil {
		t.Fatalf("ResultSummaries() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("summaries = %+v, want empty map", got)
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
