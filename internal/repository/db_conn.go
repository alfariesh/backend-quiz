package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dbConn is the minimal shape shared by *pgxpool.Pool and pgx.Tx.
type dbConn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// conn returns the tx bound to ctx if present, otherwise the pool.
func conn(ctx context.Context, pool *pgxpool.Pool) dbConn {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return pool
}
