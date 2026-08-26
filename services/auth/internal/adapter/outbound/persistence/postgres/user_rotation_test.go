package postgres

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/port/outbound"
)

const canonicalPasswordUpdateExpectation = "canonical password-only benchmark update"

func TestRotateBenchmarkPasswordsCommitsCanonicalPasswordOnlyUpdates(t *testing.T) {
	repository, mock, closeDB := newRotationRepository(t)
	defer closeDB()

	updates := benchmarkPasswordUpdates()
	mock.ExpectBegin()
	expectCanonicalPasswordUpdate(mock, updates[0], sqlmock.NewResult(0, 1))
	expectCanonicalPasswordUpdate(mock, updates[1], sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repository.RotateBenchmarkPasswords(t.Context(), updates); err != nil {
		t.Fatalf("RotateBenchmarkPasswords() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateBenchmarkPasswordsRollsBackOnDatabaseError(t *testing.T) {
	repository, mock, closeDB := newRotationRepository(t)
	defer closeDB()

	updates := benchmarkPasswordUpdates()
	mock.ExpectBegin()
	expectCanonicalPasswordUpdate(mock, updates[0], sqlmock.NewResult(0, 1))
	mock.ExpectExec(canonicalPasswordUpdateExpectation).
		WithArgs(updates[1].PasswordHash, updates[1].UserID, updates[1].Username, updates[1].Email, updates[1].FullName, rbac.RoleUser, true, false).
		WillReturnError(errors.New("database unavailable"))
	mock.ExpectRollback()

	if err := repository.RotateBenchmarkPasswords(t.Context(), updates); err == nil {
		t.Fatal("expected database error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateBenchmarkPasswordsRollsBackWhenCanonicalRowDoesNotMatch(t *testing.T) {
	repository, mock, closeDB := newRotationRepository(t)
	defer closeDB()

	updates := benchmarkPasswordUpdates()
	mock.ExpectBegin()
	expectCanonicalPasswordUpdate(mock, updates[0], sqlmock.NewResult(0, 1))
	expectCanonicalPasswordUpdate(mock, updates[1], sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err := repository.RotateBenchmarkPasswords(t.Context(), updates)
	if err == nil || !strings.Contains(err.Error(), "canonical benchmark identity changed") {
		t.Fatalf("expected canonical mismatch error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newRotationRepository(t *testing.T) (*userRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(matchCanonicalPasswordUpdate)))
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		_ = sqlDB.Close()
		t.Fatal(err)
	}
	return &userRepository{db: db}, mock, func() { _ = sqlDB.Close() }
}

func benchmarkPasswordUpdates() []outbound.BenchmarkPasswordUpdate {
	return []outbound.BenchmarkPasswordUpdate{
		{UserID: "id-051", Username: "benchmark_judge_051", Email: "benchmark-judge-051@benchmark.invalid", FullName: "Benchmark Judge 051", PasswordHash: "hash-051"},
		{UserID: "id-052", Username: "benchmark_judge_052", Email: "benchmark-judge-052@benchmark.invalid", FullName: "Benchmark Judge 052", PasswordHash: "hash-052"},
	}
}

func expectCanonicalPasswordUpdate(mock sqlmock.Sqlmock, update outbound.BenchmarkPasswordUpdate, result driver.Result) {
	mock.ExpectExec(canonicalPasswordUpdateExpectation).
		WithArgs(update.PasswordHash, update.UserID, update.Username, update.Email, update.FullName, rbac.RoleUser, true, false).
		WillReturnResult(result)
}

func matchCanonicalPasswordUpdate(expectedSQL, actualSQL string) error {
	if expectedSQL != canonicalPasswordUpdateExpectation {
		return fmt.Errorf("unexpected SQL expectation %q", expectedSQL)
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(actualSQL), " "))
	setIndex, whereIndex := strings.Index(normalized, " set "), strings.Index(normalized, " where ")
	if !strings.HasPrefix(normalized, "update ") || setIndex == -1 || whereIndex == -1 || setIndex >= whereIndex {
		return fmt.Errorf("expected UPDATE with SET and WHERE, got %q", actualSQL)
	}
	setClause := normalized[setIndex+len(" set ") : whereIndex]
	if setClause != `"password"=$1` {
		return fmt.Errorf("expected password-only SET clause, got %q", setClause)
	}
	whereClause := normalized[whereIndex+len(" where "):]
	for _, predicate := range []string{
		"id = $2",
		"username = $3",
		"email = $4",
		"full_name = $5",
		"role = $6",
		"is_active = $7",
		"is_suspended = $8",
	} {
		if !strings.Contains(whereClause, predicate) {
			return fmt.Errorf("missing canonical predicate %q in %q", predicate, actualSQL)
		}
	}
	return nil
}
