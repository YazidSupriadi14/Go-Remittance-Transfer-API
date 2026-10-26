package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"finshot/src/transfer"
)

// timeLayout is ISO-8601 in UTC with milliseconds, e.g. 2026-10-26T03:15:00.123Z
const timeLayout = "2006-01-02T15:04:05.000Z07:00"

const selectTransfer = `
	SELECT transfer_id, customer_id, idempotency_key, request_hash,
	       send_currency, send_amount, fee, total_debit,
	       receive_currency, receive_amount_minor, fx_rate,
	       recipient_name, status, created_at, updated_at
	FROM transfers`

// TransferRepository implements transfer.Repository.
type TransferRepository struct {
	db *sql.DB
}

func NewTransferRepository(db *sql.DB) *TransferRepository {
	return &TransferRepository{db: db}
}

func (r *TransferRepository) Insert(ctx context.Context, t transfer.Transfer) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO transfers (
			transfer_id, customer_id, idempotency_key, request_hash,
			send_currency, send_amount, fee, total_debit,
			receive_currency, receive_amount_minor, fx_rate,
			recipient_name, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.CustomerID, t.IdempotencyKey, t.RequestHash,
		t.SendCurrency, t.Quote.SendAmountKRW, t.Quote.FeeKRW, t.Quote.TotalDebitKRW,
		t.ReceiveCurrency, t.Quote.ReceiveAmountCentavos, t.FXRate,
		t.RecipientName, string(t.Status), formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	)
	if isDuplicateIdempotencyKey(err) {
		return transfer.ErrDuplicateIdempotencyKey
	}
	return err
}

func (r *TransferRepository) FindByID(ctx context.Context, id string) (transfer.Transfer, error) {
	row := r.db.QueryRowContext(ctx, selectTransfer+` WHERE transfer_id = ?`, id)
	return scanTransfer(row)
}

func (r *TransferRepository) FindByIdempotencyKey(ctx context.Context, key string) (transfer.Transfer, error) {
	row := r.db.QueryRowContext(ctx, selectTransfer+` WHERE idempotency_key = ?`, key)
	return scanTransfer(row)
}

func (r *TransferRepository) UpdateStatus(ctx context.Context, id string, from, to transfer.Status, at time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE transfers
		SET status = ?, updated_at = ?
		WHERE transfer_id = ? AND status = ?`,
		string(to), formatTime(at), id, string(from),
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func scanTransfer(row *sql.Row) (transfer.Transfer, error) {
	var (
		t                    transfer.Transfer
		status               string
		createdAt, updatedAt string
	)
	err := row.Scan(
		&t.ID, &t.CustomerID, &t.IdempotencyKey, &t.RequestHash,
		&t.SendCurrency, &t.Quote.SendAmountKRW, &t.Quote.FeeKRW, &t.Quote.TotalDebitKRW,
		&t.ReceiveCurrency, &t.Quote.ReceiveAmountCentavos, &t.FXRate,
		&t.RecipientName, &status, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return transfer.Transfer{}, transfer.ErrRecordNotFound
	}
	if err != nil {
		return transfer.Transfer{}, err
	}

	t.Status = transfer.Status(status)
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return transfer.Transfer{}, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return transfer.Transfer{}, err
	}
	return t, nil
}

// isDuplicateIdempotencyKey recognises the UNIQUE violation on idempotency_key only,
// so other constraint errors are never mistaken for a retry.
func isDuplicateIdempotencyKey(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE &&
		strings.Contains(sqliteErr.Error(), "transfers.idempotency_key")
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(timeLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", value, err)
	}
	return t, nil
}
