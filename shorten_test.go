package shorten

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

type mockRows struct {
	columns []string
	rows    [][]any
	pos     int
	closed  bool
}

func (m *mockRows) Columns() []string {
	return m.columns
}

func (m *mockRows) Next() bool {
	if m.pos < len(m.rows) {
		m.pos++
		return true
	}
	return false
}

func (m *mockRows) Scan(dest ...any) error {
	if m.pos == 0 || m.pos > len(m.rows) {
		return fmt.Errorf("no current row")
	}

	row := m.rows[m.pos-1]
	if len(dest) != len(row) {
		return fmt.Errorf("expected %d destinations, got %d", len(row), len(dest))
	}

	for i, value := range row {
		d := reflect.ValueOf(dest[i])
		if d.Kind() != reflect.Pointer || d.IsNil() {
			return fmt.Errorf("destination %d must be non-nil pointer", i)
		}
		elem := d.Elem()
		val := reflect.ValueOf(value)
		if !val.Type().AssignableTo(elem.Type()) {
			return fmt.Errorf("cannot assign %T to %s", value, elem.Type())
		}
		elem.Set(val)
	}
	return nil
}

func (m *mockRows) Close() error {
	m.closed = true
	return nil
}

type mockExec struct {
	queryFunc     func(ctx context.Context, query string, args ...any) (Rows, error)
	execResult    int64
	execErr       error
	execQuery     string
	execArgs      []any
	releaseCalled bool
	releaseErr    error
}

func (m *mockExec) Prepare(ctx context.Context, query string) (Stmt, error) {
	return nil, nil
}

func (m *mockExec) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	m.execQuery = query
	m.execArgs = append([]any(nil), args...)
	return m.execResult, m.execErr
}

func (m *mockExec) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return m.queryFunc(ctx, query, args...)
}

func (m *mockExec) Release(err *error) {
	m.releaseCalled = true
	if err != nil && *err == nil {
		*err = m.releaseErr
	}
}

type mockTx struct {
	commitCount   int
	rollbackCount int
	commitErr     error
	rollbackErr   error
}

func (m *mockTx) Prepare(ctx context.Context, query string) (Stmt, error) {
	return nil, nil
}

func (m *mockTx) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	return 0, nil
}

func (m *mockTx) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return nil, nil
}

func (m *mockTx) Release(*error) {
}

func (m *mockTx) Commit() error {
	m.commitCount++
	return m.commitErr
}

func (m *mockTx) Rollback() error {
	m.rollbackCount++
	return m.rollbackErr
}

type mockFactory struct {
	conn      Exec
	tx        Tx
	connCount int
	txCount   int
	connErr   error
	txErr     error
}

func (m *mockFactory) getConn(ctx context.Context) (Exec, error) {
	m.connCount++
	return m.conn, m.connErr
}

func (m *mockFactory) getTx(ctx context.Context, level sql.IsolationLevel) (Tx, error) {
	m.txCount++
	return m.tx, m.txErr
}

func TestGet_UsesTransactionScope(t *testing.T) {
	ctx, scope := Scope(context.Background())
	tx := &mockTx{}
	factory := &mockFactory{tx: tx}

	exec, err := Get(ctx, factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exec != tx {
		t.Fatalf("expected transaction from Get, got %T", exec)
	}
	if scope.tx != tx {
		t.Fatalf("expected transaction stored in scope")
	}
	if factory.txCount != 1 {
		t.Fatalf("expected getTx called once, got %d", factory.txCount)
	}
}

func TestGet_UsesConnectionWithoutScope(t *testing.T) {
	conn := &mockExec{}
	factory := &mockFactory{conn: conn}

	exec, err := Get(context.Background(), factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exec != conn {
		t.Fatalf("expected direct connection from Get, got %T", exec)
	}
	if factory.connCount != 1 {
		t.Fatalf("expected getConn called once, got %d", factory.connCount)
	}
}
