package shorten

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestScopeOptions_RequireNewAndSuppress(t *testing.T) {
	ctx := context.Background()
	ctxWithScope, scope := ScopeOptions(ctx, false, sql.LevelSerializable)
	if scope == nil {
		t.Fatal("expected scope")
	}
	if scope.level != sql.LevelSerializable {
		t.Fatalf("expected isolation level %v, got %v", sql.LevelSerializable, scope.level)
	}

	ctxWithNextScope, nextScope := ScopeOptions(ctxWithScope, true, sql.LevelRepeatableRead)
	if nextScope == nil {
		t.Fatal("expected new scope")
	}
	if nextScope == scope {
		t.Fatal("expected requireNew to create a new scope")
	}
	if got, _ := ctxWithNextScope.Value(contextTxKey).(*TxScope); got != nextScope {
		t.Fatal("expected context to have next transaction scope")
	}

	suppressed := SuppressScope(ctxWithScope)
	if suppressed == ctxWithScope {
		t.Fatal("expected suppress scope to return a new context value")
	}
	if got, _ := suppressed.Value(contextTxKey).(*TxScope); got != nil {
		t.Fatal("expected suppressed context to have nil transaction scope")
	}
}

func TestScope_NestedScopes(t *testing.T) {
	ctx := context.Background()
	ctxWithScope, scope := Scope(ctx)
	if scope == nil {
		t.Fatal("expected scope")
	}

	ctxWithNextScope, nextScope := Scope(ctxWithScope)
	if nextScope == nil {
		t.Fatal("expected new scope")
	}
	if nextScope == scope {
		t.Fatal("expected requireNew to create a new scope")
	}
	if got, _ := ctxWithNextScope.Value(contextTxKey).(*TxScope); got != scope {
		t.Fatal("expected context to have first transaction scope")
	}
}

func TestTxScope_EndWithError(t *testing.T) {
	tx := &mockTx{}
	scope := &TxScope{tx: tx}

	err := errors.New("error")
	scope.End(&err)
	if err == nil {
		t.Fatalf("unexpected not error, but got: %v", err)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("expected rollback once, got %d", tx.rollbackCount)
	}

	if scope.tx != nil {
		t.Fatal("expected scope to forgot transaction, but not")
	}
}

func TestTxScope_EndDuringPanicRollsBack(t *testing.T) {
	tx := &mockTx{}
	scope := &TxScope{tx: tx}

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()

		defer scope.End(nil)

		panic("boom")
	}()

	if !panicked {
		t.Fatal("expected panic to be propagated")
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("expected rollback once during panic, got %d", tx.rollbackCount)
	}
}

func TestTxScope_EndSetsRollbackError(t *testing.T) {
	rollbackErr := errors.New("rollback failed")
	tx := &mockTx{rollbackErr: rollbackErr}
	scope := &TxScope{tx: tx}

	scope.Rollback()
	var err error
	scope.End(&err)
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected %v, got %v", rollbackErr, err)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("expected rollback once, got %d", tx.rollbackCount)
	}
}

func TestTxScope_EndSetsCommitError(t *testing.T) {
	commitErr := errors.New("rollback failed")
	tx := &mockTx{commitErr: commitErr}
	scope := &TxScope{tx: tx}

	var err error
	scope.End(&err)
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if !errors.Is(err, commitErr) {
		t.Fatalf("expected %v, got %v", commitErr, err)
	}
	if tx.commitCount != 1 {
		t.Fatalf("expected rollback once, got %d", tx.commitCount)
	}
}

func TestTxScope_NoOverrideCurrentErrorByRollbackError(t *testing.T) {
	initialErr := errors.New("failed")
	rollbackErr := errors.New("rollback failed")
	tx := &mockTx{rollbackErr: rollbackErr}
	scope := &TxScope{tx: tx}

	err := initialErr
	scope.End(&err)
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if !errors.Is(err, initialErr) {
		t.Fatalf("expected %v, got %v", rollbackErr, err)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("expected rollback once, got %d", tx.rollbackCount)
	}
}
