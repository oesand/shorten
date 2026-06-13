package shorten

import (
	"context"
	"errors"
	"testing"
)

func TestQueryAndQuerySingle(t *testing.T) {
	exec := &mockExec{
		queryFunc: func(ctx context.Context, query string, args ...any) (Rows, error) {
			if query == "select values" {
				return &mockRows{columns: []string{"value"}, rows: [][]any{{10}, {20}}}, nil
			}
			return &mockRows{columns: []string{"value"}, rows: [][]any{}}, nil
		},
	}

	values, err := Query[int](exec, context.Background(), "select values")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 2 || values[0] != 10 || values[1] != 20 {
		t.Fatalf("unexpected query values: %#v", values)
	}

	value, err := QuerySingle[int](exec, context.Background(), "select none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 0 {
		t.Fatalf("expected zero value from QuerySingle with no rows, got %v", value)
	}
}

func TestFireExec_SuccessReleasesConnection(t *testing.T) {
	exec := &mockExec{execResult: 5}
	factory := &mockFactory{conn: exec}

	got, err := FireExec(factory, context.Background(), "update foo set bar = ?", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 5 {
		t.Fatalf("expected result 5, got %d", got)
	}
	if exec.execQuery != "update foo set bar = ?" {
		t.Fatalf("unexpected query: %q", exec.execQuery)
	}
	if len(exec.execArgs) != 1 || exec.execArgs[0] != 42 {
		t.Fatalf("unexpected exec args: %#v", exec.execArgs)
	}
	if !exec.releaseCalled {
		t.Fatal("expected Release to be called")
	}
}

func TestFireExec_FactoryError(t *testing.T) {
	errExpected := errors.New("connection failed")
	factory := &mockFactory{connErr: errExpected}

	got, err := FireExec(factory, context.Background(), "update foo")
	if got != -1 {
		t.Fatalf("expected -1 result on error, got %d", got)
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errExpected) {
		t.Fatalf("expected %v, got %v", errExpected, err)
	}
}

func TestFireExec_ReleaseSetsError(t *testing.T) {
	releaseErr := errors.New("release failed")
	exec := &mockExec{execResult: 7, releaseErr: releaseErr}
	factory := &mockFactory{conn: exec}

	got, err := FireExec(factory, context.Background(), "update foo set bar = ?", 1)
	if got != 7 {
		t.Fatalf("expected result 7, got %d", got)
	}
	if err == nil {
		t.Fatal("expected release error")
	}
	if !errors.Is(err, releaseErr) {
		t.Fatalf("expected %v, got %v", releaseErr, err)
	}
	if !exec.releaseCalled {
		t.Fatal("expected Release to be called")
	}
}

func TestFireQuery_Success(t *testing.T) {
	exec := &mockExec{
		queryFunc: func(ctx context.Context, query string, args ...any) (Rows, error) {
			return &mockRows{columns: []string{"value"}, rows: [][]any{{1}, {2}, {3}}}, nil
		},
	}
	factory := &mockFactory{conn: exec}

	values, err := FireQuery[int](factory, context.Background(), "select value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 3 || values[0] != 1 || values[2] != 3 {
		t.Fatalf("unexpected values: %#v", values)
	}
	if !exec.releaseCalled {
		t.Fatal("expected Release to be called")
	}
}

func TestFireQuerySingle_EmptyRowsReturnsZero(t *testing.T) {
	exec := &mockExec{
		queryFunc: func(ctx context.Context, query string, args ...any) (Rows, error) {
			return &mockRows{columns: []string{"value"}, rows: [][]any{}}, nil
		},
	}
	factory := &mockFactory{conn: exec}

	value, err := FireQuerySingle[int](factory, context.Background(), "select none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 0 {
		t.Fatalf("expected zero value, got %v", value)
	}
	if !exec.releaseCalled {
		t.Fatal("expected Release to be called")
	}
}
