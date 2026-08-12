package postgres

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"
)

func profileStatsRows(columns []string, values ...[]driver.Value) driver.Rows {
	return &repositoryReadRows{columns: columns, values: values}
}

func TestSubmissionRepositoryGetUserProfileStatsUsesBoundedAggregates(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")
	activitySince := time.Date(2025, 8, 13, 0, 0, 0, 0, time.UTC)
	var queries []string
	var contextValues []any
	var queryArgs [][]driver.NamedValue
	db := newRepositoryReadGormDB(t, func(queryCtx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		queries = append(queries, query)
		contextValues = append(contextValues, queryCtx.Value(contextKey{}))
		queryArgs = append(queryArgs, args)
		switch len(queries) {
		case 1:
			return profileStatsRows(
				[]string{"total_submissions", "attempted_problems", "accepted_submissions", "solved_problems"},
				[]driver.Value{int64(4), int64(3), int64(2), int64(2)},
			), nil
		case 2:
			return profileStatsRows([]string{"verdict", "count"},
				[]driver.Value{"ACCEPTED", int64(2)},
				[]driver.Value{"WRONG_ANSWER", int64(1)},
			), nil
		case 3:
			return profileStatsRows([]string{"language", "count"},
				[]driver.Value{"GO", int64(3)},
				[]driver.Value{"CPP", int64(1)},
			), nil
		default:
			return profileStatsRows([]string{"date", "count"}, []driver.Value{"2026-08-12", int64(3)}), nil
		}
	})

	got, err := (&submissionRepository{db: db}).GetUserProfileStats(ctx, "actor", activitySince)
	if err != nil {
		t.Fatalf("GetUserProfileStats() error = %v", err)
	}
	if got.TotalSubmissions != 4 || got.AttemptedProblems != 3 || got.AcceptedSubmissions != 2 || got.SolvedProblems != 2 ||
		len(got.Verdicts) != 2 || len(got.Languages) != 2 || len(got.Activity) != 1 {
		t.Fatalf("stats = %+v", got)
	}
	if len(queries) != 4 {
		t.Fatalf("query count = %d, want 4", len(queries))
	}
	for index, value := range contextValues {
		if value != "request-context" {
			t.Fatalf("query %d context value = %v", index, value)
		}
	}

	summary := strings.ToUpper(queries[0])
	for _, clause := range []string{"COUNT(*)", "COUNT(DISTINCT PROBLEM_ID) AS ATTEMPTED_PROBLEMS", "FILTER (WHERE STATUS", "USER_ID ="} {
		if !strings.Contains(summary, clause) {
			t.Fatalf("summary query missing %q: %s", clause, queries[0])
		}
	}
	if strings.Contains(summary, "SUBMISSION_RESULTS") || strings.Contains(summary, "SUBMISSION_ATTEMPTS") {
		t.Fatalf("summary must aggregate current submission rows only: %s", queries[0])
	}
	for index, args := range queryArgs {
		if !containsDriverValue(args, "actor") {
			t.Fatalf("query %d does not contain authenticated user ID: %+v", index, args)
		}
	}

	verdicts := strings.ToUpper(queries[1])
	if !strings.Contains(verdicts, "STATUS IN") || !strings.Contains(verdicts, "ORDER BY COUNT DESC, STATUS ASC") {
		t.Fatalf("verdict query = %s", queries[1])
	}
	if containsDriverValue(queryArgs[1], "PENDING") || containsDriverValue(queryArgs[1], "JUDGING") {
		t.Fatalf("terminal verdict query includes an unfinished state: %+v", queryArgs[1])
	}
	languages := strings.ToUpper(queries[2])
	if !strings.Contains(languages, "GROUP BY LANGUAGE") || !strings.Contains(languages, "ORDER BY COUNT DESC, LANGUAGE ASC") {
		t.Fatalf("language query = %s", queries[2])
	}
	activity := strings.ToUpper(queries[3])
	if !strings.Contains(activity, "CREATED_AT >=") || !strings.Contains(activity, "AT TIME ZONE 'UTC'") || !strings.Contains(activity, "ORDER BY") {
		t.Fatalf("activity query = %s", queries[3])
	}
}

func TestSubmissionRepositoryGetUserProfileStatsWrapsQueryError(t *testing.T) {
	db := newTestGormDB(t)
	db.AddError(context.DeadlineExceeded)
	_, err := (&submissionRepository{db: db}).GetUserProfileStats(context.Background(), "actor", time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "get user profile stats summary") {
		t.Fatalf("error = %v", err)
	}
}
