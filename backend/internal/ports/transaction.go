package ports

import "context"

// TransactionManager provides transaction boundaries without exposing a
// database driver or ORM type to the application layer.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}
