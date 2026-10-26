package transfer

import (
	"context"
	"time"
)

// The service depends on these small interfaces, not on SQLite.
// Any storage (SQLite, Postgres, an in-memory fake for tests) can be plugged in.

// Repository stores transfers.
type Repository interface {
	// Insert saves a new transfer.
	// It returns ErrDuplicateIdempotencyKey if the idempotency key is already used.
	Insert(ctx context.Context, t Transfer) error

	// FindByID returns ErrRecordNotFound if no transfer has this ID.
	FindByID(ctx context.Context, id string) (Transfer, error)

	// FindByIdempotencyKey returns ErrRecordNotFound if the key was never used.
	FindByIdempotencyKey(ctx context.Context, key string) (Transfer, error)

	// UpdateStatus changes the status only if the current status is still `from`.
	// It returns false when the row was not changed (status moved on in the meantime).
	UpdateStatus(ctx context.Context, id string, from, to Status, at time.Time) (bool, error)
}

// CustomerDirectory answers whether a customer exists.
type CustomerDirectory interface {
	Exists(ctx context.Context, customerID string) (bool, error)
}
