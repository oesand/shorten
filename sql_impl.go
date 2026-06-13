package shorten

import (
	"context"
	"database/sql"
)

func SqlFactory(db *sql.DB) Factory {
	return &sqlFactory{db}
}

type sqlFactory struct {
	db *sql.DB
}

func (s *sqlFactory) getConn(ctx context.Context) (Exec, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	return &sqlConn{conn}, nil
}

func (s *sqlFactory) getTx(ctx context.Context, level sql.IsolationLevel) (Tx, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: level})
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx}, nil
}

type sqlRows struct {
	*sql.Rows
}

func (s *sqlRows) Columns() []string {
	columns, _ := s.Rows.Columns()
	return columns
}

func (s *sqlRows) Next() bool {
	return s.Rows.Next()
}

func (s *sqlRows) Scan(dest ...any) error {
	return s.Rows.Scan(dest...)
}

func (s *sqlRows) Close() error {
	return s.Rows.Close()
}

type sqlStmt struct {
	*sql.Stmt
}

func (s *sqlStmt) Exec(ctx context.Context, args ...any) (int64, error) {
	result, err := s.Stmt.ExecContext(ctx, args...)
	if err != nil {
		return -1, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func (s *sqlStmt) Query(ctx context.Context, args ...any) (Rows, error) {
	rows, err := s.Stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}

	return &sqlRows{rows}, nil
}

type sqlConn struct {
	*sql.Conn
}

func (s *sqlConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	stmt, err := s.Conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &sqlStmt{stmt}, nil
}

func (s *sqlConn) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := s.Conn.ExecContext(ctx, query, args...)
	if err != nil {
		return -1, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func (s *sqlConn) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := s.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return &sqlRows{rows}, nil
}

func (s *sqlConn) Release(err *error) {
	cerr := s.Conn.Close()
	if err != nil && *err == nil {
		*err = cerr
	}
}

type sqlTx struct {
	*sql.Tx
}

func (s *sqlTx) Prepare(ctx context.Context, query string) (Stmt, error) {
	stmt, err := s.Tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &sqlStmt{stmt}, nil
}

func (s *sqlTx) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := s.Tx.ExecContext(ctx, query, args...)
	if err != nil {
		return -1, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func (s *sqlTx) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := s.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return &sqlRows{rows}, nil
}

func (s *sqlTx) Release(*error) {
}
