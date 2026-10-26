package transfer_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"finshot/src/storage"
	"finshot/src/transfer"
)

// newTestService wires the real service to a real in-memory SQLite database,
// so the tests also prove that the database constraints do their job.
func newTestService(t *testing.T) *transfer.Service {
	t.Helper()
	db, err := storage.Open("", "../../schema.sql")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return transfer.NewService(
		storage.NewTransferRepository(db),
		storage.NewCustomerRepository(db),
		time.Now,
		uuid.NewString,
	)
}

func validCommand(key string) transfer.CreateCommand {
	return transfer.CreateCommand{
		IdempotencyKey:  key,
		CustomerID:      "C001",
		SendCurrency:    "KRW",
		SendAmount:      "500000",
		ReceiveCurrency: "PHP",
		RecipientName:   "Juan Dela Cruz",
	}
}

// Many identical requests arrive at the same moment: exactly one transfer must exist.
func TestCreate_ConcurrentRetriesCreateOnlyOneTransfer(t *testing.T) {
	service := newTestService(t)
	const requests = 20

	var (
		wg      sync.WaitGroup
		results = make([]transfer.CreateResult, requests)
		errs    = make([]error, requests)
	)
	for i := range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = service.Create(context.Background(), validCommand("same-key"))
		}()
	}
	wg.Wait()

	created := 0
	for i := range requests {
		if errs[i] != nil {
			t.Fatalf("request %d failed: %v", i, errs[i])
		}
		if results[i].Transfer.ID != results[0].Transfer.ID {
			t.Fatalf("request %d got transfer %s, want %s", i, results[i].Transfer.ID, results[0].Transfer.ID)
		}
		if !results[i].Replayed {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("%d requests created a transfer, want exactly 1", created)
	}
}

// The same key with different data is a conflict, and the original transfer is untouched.
func TestCreate_SameKeyDifferentDataIsConflict(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	original, err := service.Create(ctx, validCommand("key-1"))
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	changed := validCommand("key-1")
	changed.SendAmount = "600000"
	_, err = service.Create(ctx, changed)

	var businessErr *transfer.Error
	if !errors.As(err, &businessErr) || businessErr.Kind != transfer.KindConflict {
		t.Fatalf("got error %v, want a conflict", err)
	}

	stored, err := service.Get(ctx, original.Transfer.ID)
	if err != nil {
		t.Fatalf("get original: %v", err)
	}
	if stored.Quote.SendAmountKRW != 500_000 {
		t.Fatalf("original sendAmount changed to %d", stored.Quote.SendAmountKRW)
	}
}
