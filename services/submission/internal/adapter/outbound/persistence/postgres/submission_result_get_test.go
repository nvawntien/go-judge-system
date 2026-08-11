package postgres

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"go-judge-system/services/submission/internal/domain/entity"
)

func submissionResultRows(values ...[]driver.Value) driver.Rows {
	return &repositoryReadRows{
		columns: []string{
			"id",
			"submission_id",
			"attempt_id",
			"test_index",
			"status",
			"actual_output",
			"input",
			"expected_output",
			"execution_time",
			"memory_used",
			"created_at",
		},
		values: values,
	}
}

func TestSubmissionResultRepositoryGetBySubmissionIDAndAttemptID(t *testing.T) {
	createdAt := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")

	queryCalls := 0
	var gotQuery string
	var gotArgs []driver.NamedValue
	var gotContextValue any
	db := newRepositoryReadGormDB(t, func(
		queryCtx context.Context,
		query string,
		args []driver.NamedValue,
	) (driver.Rows, error) {
		queryCalls++
		gotQuery = query
		gotArgs = append([]driver.NamedValue(nil), args...)
		gotContextValue = queryCtx.Value(contextKey{})
		return submissionResultRows([]driver.Value{
			int64(1),
			int64(77),
			"attempt-current",
			int64(3),
			"ACCEPTED",
			"hidden stdout",
			"hidden input",
			"hidden expected",
			int64(2),
			int64(128),
			createdAt,
		}), nil
	})

	got, err := (&submissionResultRepository{db: db}).GetBySubmissionIDAndAttemptID(ctx, 77, " attempt-current ")
	if err != nil {
		t.Fatalf("GetBySubmissionIDAndAttemptID() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("result count = %d, want 1", len(got))
	}
	if got[0].SubmissionID != 77 ||
		got[0].AttemptID != "attempt-current" ||
		got[0].TestIndex != 3 ||
		got[0].Status != entity.ResultAccepted {
		t.Fatalf("result = %+v", got[0])
	}
	if queryCalls != 1 {
		t.Fatalf("query calls = %d, want 1", queryCalls)
	}
	if gotContextValue != "request-context" {
		t.Fatalf("query context value = %v, want propagated value", gotContextValue)
	}
	upperQuery := strings.ToUpper(gotQuery)
	for _, required := range []string{"SUBMISSION_ID", "ATTEMPT_ID", "ORDER BY"} {
		if !strings.Contains(upperQuery, required) {
			t.Fatalf("query missing %q: %s", required, gotQuery)
		}
	}
	if len(gotArgs) != 2 || gotArgs[0].Value != int64(77) || gotArgs[1].Value != "attempt-current" {
		t.Fatalf("query args = %+v, want submission ID and trimmed attempt ID", gotArgs)
	}
}

func TestSubmissionResultRepositoryGetBySubmissionIDAndAttemptIDSkipsBlankAttempt(t *testing.T) {
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		t.Fatal("database query must not run for blank attempt ID")
		return nil, nil
	})

	got, err := (&submissionResultRepository{db: db}).GetBySubmissionIDAndAttemptID(context.Background(), 77, " ")
	if err != nil {
		t.Fatalf("GetBySubmissionIDAndAttemptID() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("results = %#v, want non-nil empty slice", got)
	}
}
