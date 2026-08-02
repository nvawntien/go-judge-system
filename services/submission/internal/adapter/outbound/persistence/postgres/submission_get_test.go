package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
)

type repositoryReadConnector struct {
	query func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
}

func (c *repositoryReadConnector) Connect(context.Context) (driver.Conn, error) {
	return &repositoryReadConn{query: c.query}, nil
}

func (*repositoryReadConnector) Driver() driver.Driver {
	return repositoryReadDriver{}
}

type repositoryReadDriver struct{}

func (repositoryReadDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("repository read driver requires a connector")
}

type repositoryReadConn struct {
	query func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
}

func (*repositoryReadConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (*repositoryReadConn) Close() error {
	return nil
}

func (*repositoryReadConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c *repositoryReadConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	return c.query(ctx, query, args)
}

type repositoryReadRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *repositoryReadRows) Columns() []string {
	return r.columns
}

func (*repositoryReadRows) Close() error {
	return nil
}

func (r *repositoryReadRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func newRepositoryReadGormDB(
	t *testing.T,
	query func(context.Context, string, []driver.NamedValue) (driver.Rows, error),
) *gorm.DB {
	t.Helper()

	sqlDB := sql.OpenDB(&repositoryReadConnector{query: query})
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	db, err := gorm.Open(testDialector{pool: sqlDB}, &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("open test GORM database: %v", err)
	}
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return db
}

func submissionRows(values ...[]driver.Value) driver.Rows {
	return &repositoryReadRows{
		columns: []string{
			"id",
			"problem_id",
			"problem_name",
			"user_id",
			"username",
			"language",
			"source_code",
			"current_attempt_id",
			"status",
			"execution_time",
			"memory_used",
			"compile_output",
			"error_message",
			"created_at",
			"updated_at",
		},
		values: values,
	}
}

func TestSubmissionRepositoryGetByID(t *testing.T) {
	createdAt := time.Date(2026, 7, 23, 14, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 7, 23, 14, 1, 0, 0, time.UTC)
	errorMessage := "panic: runtime error: index out of range"
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")

	queryCalls := 0
	var gotQuery string
	var gotContextValue any
	db := newRepositoryReadGormDB(t, func(
		queryCtx context.Context,
		query string,
		_ []driver.NamedValue,
	) (driver.Rows, error) {
		queryCalls++
		gotQuery = query
		gotContextValue = queryCtx.Value(contextKey{})
		return submissionRows([]driver.Value{
			int64(77),
			int64(42),
			"Two Sum",
			"owner",
			"owner-name",
			"GO",
			"package main\n",
			"attempt-77",
			"PENDING",
			nil,
			nil,
			nil,
			errorMessage,
			createdAt,
			updatedAt,
		}), nil
	})

	got, err := (&submissionRepository{db: db}).GetByID(ctx, 77)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	want := &entity.Submission{
		ID:               77,
		ProblemID:        42,
		ProblemName:      "Two Sum",
		UserID:           "owner",
		Username:         "owner-name",
		Language:         entity.LanguageGo,
		SourceCode:       "package main\n",
		CurrentAttemptID: "attempt-77",
		Status:           entity.StatusPending,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != errorMessage {
		t.Fatalf("error message = %v, want %q", got.ErrorMessage, errorMessage)
	}
	got.ErrorMessage = nil
	if *got != *want {
		t.Fatalf("submission = %+v, want %+v", got, want)
	}
	if queryCalls != 1 {
		t.Fatalf("query calls = %d, want 1", queryCalls)
	}
	if gotContextValue != "request-context" {
		t.Fatalf("query context value = %v, want propagated value", gotContextValue)
	}
	if strings.Contains(strings.ToUpper(gotQuery), "JOIN") {
		t.Fatalf("GetByID query unexpectedly joins unrelated data: %s", gotQuery)
	}
}

func TestSubmissionRepositoryGetByIDNotFound(t *testing.T) {
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		return submissionRows(), nil
	})

	got, err := (&submissionRepository{db: db}).GetByID(context.Background(), 404)
	if got != nil {
		t.Fatalf("submission = %+v, want nil", got)
	}
	if !errors.Is(err, domain.ErrSubmissionNotFound) {
		t.Fatalf("GetByID() error = %v, want submission not found", err)
	}
}

func TestSubmissionRepositoryGetByIDWrapsDatabaseError(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		return nil, databaseErr
	})

	_, err := (&submissionRepository{db: db}).GetByID(context.Background(), 77)
	if !errors.Is(err, databaseErr) {
		t.Fatalf("GetByID() error = %v, want wrapped database error", err)
	}
	if !strings.Contains(err.Error(), "get submission 77") {
		t.Fatalf("GetByID() error = %q, want safe operation context", err)
	}
}

func TestSubmissionRepositoryGetByIDPreservesCancellation(t *testing.T) {
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

	_, err := (&submissionRepository{db: db}).GetByID(ctx, 77)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByID() error = %v, want context canceled", err)
	}
}

func submissionStreamSnapshotRows(values ...[]driver.Value) driver.Rows {
	return &repositoryReadRows{
		columns: []string{
			"id",
			"user_id",
			"current_attempt_id",
			"status",
			"updated_at",
		},
		values: values,
	}
}

func TestSubmissionRepositoryGetStreamSnapshot(t *testing.T) {
	updatedAt := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-context")

	queryCalls := 0
	var gotQuery string
	var gotContextValue any
	db := newRepositoryReadGormDB(t, func(
		queryCtx context.Context,
		query string,
		_ []driver.NamedValue,
	) (driver.Rows, error) {
		queryCalls++
		gotQuery = query
		gotContextValue = queryCtx.Value(contextKey{})
		return submissionStreamSnapshotRows([]driver.Value{
			int64(77),
			"owner",
			"attempt-77",
			"JUDGING",
			updatedAt,
		}), nil
	})

	got, err := (&submissionRepository{db: db}).GetStreamSnapshot(ctx, 77)
	if err != nil {
		t.Fatalf("GetStreamSnapshot() error = %v", err)
	}
	want := &entity.SubmissionStreamSnapshot{
		SubmissionID: 77,
		UserID:       "owner",
		AttemptID:    "attempt-77",
		Status:       entity.StatusJudging,
		UpdatedAt:    updatedAt,
	}
	if *got != *want {
		t.Fatalf("snapshot = %+v, want %+v", got, want)
	}
	if queryCalls != 1 {
		t.Fatalf("query calls = %d, want 1", queryCalls)
	}
	if gotContextValue != "request-context" {
		t.Fatalf("query context value = %v, want propagated value", gotContextValue)
	}
	upperQuery := strings.ToUpper(gotQuery)
	for _, forbidden := range []string{"SOURCE_CODE", "COMPILE_OUTPUT", "ERROR_MESSAGE"} {
		if strings.Contains(upperQuery, forbidden) {
			t.Fatalf("snapshot query selects hidden/detail field %q: %s", forbidden, gotQuery)
		}
	}
}

func TestSubmissionRepositoryGetStreamSnapshotNotFound(t *testing.T) {
	db := newRepositoryReadGormDB(t, func(
		context.Context,
		string,
		[]driver.NamedValue,
	) (driver.Rows, error) {
		return submissionStreamSnapshotRows(), nil
	})

	got, err := (&submissionRepository{db: db}).GetStreamSnapshot(context.Background(), 404)
	if got != nil {
		t.Fatalf("snapshot = %+v, want nil", got)
	}
	if !errors.Is(err, domain.ErrSubmissionNotFound) {
		t.Fatalf("GetStreamSnapshot() error = %v, want submission not found", err)
	}
}
