package transfer

import "time"

// Transfer is one remittance from a customer in Korea to a recipient in the Philippines.
type Transfer struct {
	ID             string
	CustomerID     string
	IdempotencyKey string
	RequestHash    string

	SendCurrency    string
	ReceiveCurrency string
	Quote           Quote
	FXRate          string

	RecipientName string
	Status        Status

	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateCommand is everything a client sends to create a transfer.
// Values are raw, exactly as received; Service.Create validates them.
type CreateCommand struct {
	IdempotencyKey  string
	CustomerID      string
	SendCurrency    string
	SendAmount      string
	ReceiveCurrency string
	RecipientName   string
}
