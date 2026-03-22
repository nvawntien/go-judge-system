package postgres

import (
	"context"

	"go-judge-system/services/submission/internal/application/port/outbound"

	"gorm.io/gorm"
)

type txKey struct{}

type transactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) outbound.TransactionManager {
	return &transactionManager{db: db}
}

func (tm *transactionManager) ExecuteInTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// getDB is a helper for repositories to retrieve a transaction from context if it exists,
// otherwise it returns the default DB connection.
func getDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return defaultDB.WithContext(ctx)
}
