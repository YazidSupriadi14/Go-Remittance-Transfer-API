package transfer

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Service holds the business rules for transfers.
// It talks to storage only through the Repository and CustomerDirectory interfaces.
type Service struct {
	transfers Repository
	customers CustomerDirectory
	now       func() time.Time
	newID     func() string
}

func NewService(transfers Repository, customers CustomerDirectory, now func() time.Time, newID func() string) *Service {
	return &Service{transfers: transfers, customers: customers, now: now, newID: newID}
}

// CreateResult tells the caller whether a new transfer was created or an earlier one replayed.
type CreateResult struct {
	Transfer Transfer
	Replayed bool
}

// Create creates a transfer, or returns the original one if this is a retry.
//
// How a retry is recognised:
//  1. Same Idempotency-Key + same data      -> return the original transfer.
//  2. Same Idempotency-Key + different data -> conflict.
//  3. Two identical requests at the same time: both may miss step 1, but the database
//     UNIQUE constraint lets only one INSERT win. The loser falls back to step 1.
func (s *Service) Create(ctx context.Context, cmd CreateCommand) (CreateResult, error) {
	input, err := cmd.validate()
	if err != nil {
		return CreateResult{}, err
	}
	if err := s.ensureCustomerExists(ctx, input.CustomerID); err != nil {
		return CreateResult{}, err
	}

	hash := input.requestHash()

	original, err := s.findOriginal(ctx, input.IdempotencyKey, hash)
	if err == nil {
		return CreateResult{Transfer: original, Replayed: true}, nil
	}
	if !errors.Is(err, ErrRecordNotFound) {
		return CreateResult{}, err
	}

	transfer := s.newTransfer(input, hash)
	err = s.transfers.Insert(ctx, transfer)
	if errors.Is(err, ErrDuplicateIdempotencyKey) {
		// A concurrent request with the same key was saved first.
		original, err := s.findOriginal(ctx, input.IdempotencyKey, hash)
		if err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Transfer: original, Replayed: true}, nil
	}
	if err != nil {
		return CreateResult{}, fmt.Errorf("save transfer: %w", err)
	}

	return CreateResult{Transfer: transfer}, nil
}

// Get returns one transfer by ID.
func (s *Service) Get(ctx context.Context, id string) (Transfer, error) {
	transfer, err := s.transfers.FindByID(ctx, id)
	if errors.Is(err, ErrRecordNotFound) {
		return Transfer{}, notFoundError("transfer %s not found", id)
	}
	if err != nil {
		return Transfer{}, fmt.Errorf("find transfer: %w", err)
	}
	return transfer, nil
}

// Cancel moves a REQUESTED transfer to CANCELLED.
func (s *Service) Cancel(ctx context.Context, id string) (Transfer, error) {
	return s.changeStatus(ctx, id, StatusCancelled)
}

// changeStatus applies one state machine transition.
// The UPDATE only succeeds if the status in the database is still the one we read,
// so two concurrent requests can never both move the same transfer.
func (s *Service) changeStatus(ctx context.Context, id string, next Status) (Transfer, error) {
	transfer, err := s.Get(ctx, id)
	if err != nil {
		return Transfer{}, err
	}
	if !transfer.Status.CanTransitionTo(next) {
		return Transfer{}, conflictError("transfer is %s and cannot move to %s", transfer.Status, next)
	}

	now := s.currentTime()
	changed, err := s.transfers.UpdateStatus(ctx, id, transfer.Status, next, now)
	if err != nil {
		return Transfer{}, fmt.Errorf("update transfer status: %w", err)
	}
	if !changed {
		return Transfer{}, conflictError("transfer %s was changed by another request, try again", id)
	}

	transfer.Status = next
	transfer.UpdatedAt = now
	return transfer, nil
}

// findOriginal looks up the transfer created with this idempotency key.
// It returns ErrRecordNotFound if the key is new, or a conflict if the data differs.
func (s *Service) findOriginal(ctx context.Context, key, hash string) (Transfer, error) {
	original, err := s.transfers.FindByIdempotencyKey(ctx, key)
	if err != nil {
		return Transfer{}, err
	}
	if original.RequestHash != hash {
		return Transfer{}, conflictError("Idempotency-Key %q was already used with different request data", key)
	}
	return original, nil
}

func (s *Service) ensureCustomerExists(ctx context.Context, customerID string) error {
	exists, err := s.customers.Exists(ctx, customerID)
	if err != nil {
		return fmt.Errorf("check customer: %w", err)
	}
	if !exists {
		return validationError("unknown customerId %s", customerID)
	}
	return nil
}

func (s *Service) newTransfer(input validCreateInput, hash string) Transfer {
	now := s.currentTime()
	return Transfer{
		ID:              s.newID(),
		CustomerID:      input.CustomerID,
		IdempotencyKey:  input.IdempotencyKey,
		RequestHash:     hash,
		SendCurrency:    input.SendCurrency,
		ReceiveCurrency: input.ReceiveCurrency,
		Quote:           NewQuote(input.SendAmountKRW),
		FXRate:          FXRate,
		RecipientName:   input.RecipientName,
		Status:          StatusRequested,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// currentTime is truncated to milliseconds, the precision we store and return.
// This keeps a freshly created transfer identical to the same transfer read back later.
func (s *Service) currentTime() time.Time {
	return s.now().UTC().Truncate(time.Millisecond)
}
