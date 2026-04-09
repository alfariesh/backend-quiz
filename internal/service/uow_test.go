package service

import "context"

// noopUoW is a pass-through UnitOfWork for tests — just calls fn directly.
type noopUoW struct{}

func (noopUoW) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
