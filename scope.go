package shorten

import (
	"context"
	"database/sql"
)

var contextTxKey = ctxKey{Key: "shorten/tx"}

var DefaultLevel = sql.LevelDefault

// Scope returns a context containing a transaction scope. It reuses an existing
// scope when one is already present in the context.
func Scope(ctx context.Context) (context.Context, *TxScope) {
	return ScopeOptions(ctx, false, DefaultLevel)
}

// ScopeOptions returns a context containing a transaction scope with the given
// isolation level. If requireNew is true, a fresh scope is created regardless of
// any existing scope in the context.
func ScopeOptions(ctx context.Context, requireNew bool, level sql.IsolationLevel) (context.Context, *TxScope) {
	scope := &TxScope{
		level: level,
	}

	parent, _ := ctx.Value(contextTxKey).(*TxScope)
	if requireNew || parent == nil {
		ctx = context.WithValue(ctx, contextTxKey, scope)
	}

	return ctx, scope
}

// SuppressScope returns a context that explicitly removes any active transaction scope.
func SuppressScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextTxKey, nil)
}

// TxScope controls rollback/rollback behavior for a transaction lifecycle stored in context.
type TxScope struct {
	level sql.IsolationLevel

	tx       Tx
	rollback bool
}

// Rollback marks the current transaction scope so that End will be rollback.
func (scope *TxScope) Rollback() {
	scope.rollback = true
}

// End finalizes the scoped transaction. If Commit was called, the transaction
// is committed; otherwise it is rolled back. If End is called during a panic,
// the transaction is rolled back and the panic is rethrown.
func (scope *TxScope) End(err *error) {
	if scope.tx == nil {
		return
	}

	if r := recover(); r != nil {
		_ = scope.tx.Rollback()
		scope.tx = nil
		panic(r)
	}

	if err != nil && *err != nil {
		_ = scope.tx.Rollback()
	} else {
		var ierr error
		if scope.rollback {
			ierr = scope.tx.Rollback()
		} else {
			ierr = scope.tx.Commit()
		}

		if err != nil {
			*err = ierr
		}
	}

	scope.tx = nil
}
