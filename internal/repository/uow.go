package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/repository/sqlc"
)

type ctxKey int

const txCtxKey ctxKey = iota

// TxFromContext returns the pgx.Tx stored in ctx, or nil.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txCtxKey).(pgx.Tx)
	return tx
}

// ContextWithTx returns a child context carrying the given transaction.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey, tx)
}

// querier returns a *sqlc.Queries bound to the transaction in ctx,
// or the original Queries if no transaction is active.
func querier(q *sqlc.Queries, ctx context.Context) *sqlc.Queries {
	if tx := TxFromContext(ctx); tx != nil {
		return q.WithTx(tx)
	}
	return q
}

// beginOrJoin starts a new transaction if none exists in ctx,
// or returns the existing one. The bool indicates whether a new tx was created
// (and therefore the caller is responsible for commit/rollback).
func beginOrJoin(ctx context.Context, pool *pgxpool.Pool) (context.Context, pgx.Tx, bool, error) {
	if tx := TxFromContext(ctx); tx != nil {
		return ctx, tx, false, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return ctx, nil, false, err
	}
	return ContextWithTx(ctx, tx), tx, true, nil
}

// PgxUnitOfWork implements domain.UnitOfWork using pgxpool transactions.
type PgxUnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *PgxUnitOfWork {
	return &PgxUnitOfWork{pool: pool}
}

func (u *PgxUnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// If already inside a UoW, reuse the outer transaction.
	if TxFromContext(ctx) != nil {
		return fn(ctx)
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(ContextWithTx(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
