package shorten

import (
	"context"
	"database/sql"
)

type ctxKey struct {
	Key string
}

// Rows is a minimal result set abstraction used by shorten helpers.
// It mirrors the small subset of database/sql rows operations needed for mapping.
type Rows interface {
	Columns() []string
	Next() bool
	Scan(dest ...any) error
	Close() error
}

// Stmt is a simplified prepared statement interface used by the shorten package.
type Stmt interface {
	Exec(ctx context.Context, args ...any) (int64, error)
	Query(ctx context.Context, args ...any) (Rows, error)
	Close() error
}

// Exec represents a connection or transaction that can execute statements,
// query rows, and prepare or release resources.
type Exec interface {
	Prepare(ctx context.Context, query string) (Stmt, error)
	Exec(ctx context.Context, query string, args ...any) (int64, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	Release(*error)
}

// Tx represents a transaction-capable Exec which can be committed or rolled back.
type Tx interface {
	Exec
	Commit() error
	Rollback() error
}

// Factory provides connection and transaction acquisition for Get.
// The transaction returned from getTx is used when a transaction scope exists in context.
type Factory interface {
	getConn(ctx context.Context) (Exec, error)
	getTx(ctx context.Context, level sql.IsolationLevel) (Tx, error)
}

// Get returns the active transaction from context when one exists, otherwise it
// acquires a direct connection from the factory.
func Get(ctx context.Context, factory Factory) (Exec, error) {
	scope, _ := ctx.Value(contextTxKey).(*TxScope)
	if scope != nil {
		if exec := scope.tx; exec != nil {
			return exec, nil
		}

		tx, err := factory.getTx(ctx, scope.level)
		if err != nil {
			return nil, err
		}
		scope.tx = tx
		return tx, nil
	}

	return factory.getConn(ctx)
}
