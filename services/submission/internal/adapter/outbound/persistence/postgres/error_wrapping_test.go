package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type testDialector struct {
	pool gorm.ConnPool
}

func (d testDialector) Name() string { return "test" }
func (d testDialector) Initialize(db *gorm.DB) error {
	db.ConnPool = d.pool
	return nil
}
func (testDialector) Migrator(*gorm.DB) gorm.Migrator                { return nil }
func (testDialector) DataTypeOf(*schema.Field) string                { return "" }
func (testDialector) DefaultValueOf(*schema.Field) clause.Expression { return nil }
func (testDialector) BindVarTo(clause.Writer, *gorm.Statement, interface{}) {
}
func (testDialector) QuoteTo(writer clause.Writer, value string) {
	writer.WriteString(value)
}
func (testDialector) Explain(query string, _ ...interface{}) string { return query }

type testConnPool struct{}

func (*testConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*testConnPool) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return driver.RowsAffected(0), nil
}
func (*testConnPool) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, io.EOF
}
func (*testConnPool) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return &sql.Row{}
}

type testConnectionPool struct{ testConnPool }

func (*testConnectionPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	return &testTransactionPool{}, nil
}

type testTransactionPool struct{ testConnPool }

func (*testTransactionPool) Commit() error   { return nil }
func (*testTransactionPool) Rollback() error { return nil }

func newTestGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(testDialector{pool: &testConnectionPool{}}, &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("open test GORM database: %v", err)
	}
	return db
}

func TestTransactionManager_WrapsCallbackError(t *testing.T) {
	db := newTestGormDB(t)
	txManager := NewTransactionManager(db)
	wantErr := errors.New("callback failed")

	err := txManager.ExecuteInTx(context.Background(), func(txCtx context.Context) error {
		if got := getDB(txCtx, db); got == db {
			t.Fatal("callback did not receive transaction-scoped database")
		}
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "execute transaction") {
		t.Fatalf("error = %q, want transaction operation context", err)
	}
}

func TestActiveRepositories_WrapErrors(t *testing.T) {
	wantErr := errors.New("database failed")

	t.Run("create submission", func(t *testing.T) {
		db := newTestGormDB(t)
		db.AddError(wantErr)
		repo := &submissionRepository{db: db}
		err := repo.Create(context.Background(), entity.NewSubmission(1, "", "u", "alice", entity.LanguageGo, "source"))
		assertWrappedOperation(t, err, wantErr, "create submission")
	})

	t.Run("create outbox message", func(t *testing.T) {
		db := newTestGormDB(t)
		db.AddError(wantErr)
		repo := &outboxRepository{db: db}
		err := repo.Create(context.Background(), &entity.OutboxMessage{})
		assertWrappedOperation(t, err, wantErr, "create outbox message")
	})
}

func assertWrappedOperation(t *testing.T, err, wantErr error, operation string) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), operation) {
		t.Fatalf("error = %q, want operation %q", err, operation)
	}
}
