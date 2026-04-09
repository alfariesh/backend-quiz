package domain

import "context"

// UnitOfWork wraps a function in a database transaction.
// If the function returns nil, the transaction is committed.
// If the function returns an error or panics, it is rolled back.
// Nested calls (UoW inside UoW) reuse the outer transaction.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
