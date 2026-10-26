package storage

import (
	"context"
	"database/sql"
)

// CustomerRepository implements transfer.CustomerDirectory.
type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Exists(ctx context.Context, customerID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM customers WHERE customer_id = ?)`, customerID,
	).Scan(&exists)
	return exists, err
}
