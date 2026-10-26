package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxIdempotencyKeyLength = 64
	maxRecipientNameLength  = 100
)

var (
	wholeNumberPattern   = regexp.MustCompile(`^-?[0-9]+$`)
	decimalNumberPattern = regexp.MustCompile(`^-?[0-9]+\.[0-9]+$`)
)

// validCreateInput is a CreateCommand that passed every format rule.
// Whether the customer exists is checked by the service, because it needs the database.
type validCreateInput struct {
	IdempotencyKey  string
	CustomerID      string
	SendCurrency    string
	SendAmountKRW   int64
	ReceiveCurrency string
	RecipientName   string
}

// validate checks the command and returns the first problem found.
func (c CreateCommand) validate() (validCreateInput, error) {
	key := strings.TrimSpace(c.IdempotencyKey)
	if key == "" {
		return validCreateInput{}, validationError("Idempotency-Key header is required")
	}
	if len(key) > maxIdempotencyKeyLength {
		return validCreateInput{}, validationError("Idempotency-Key must be at most %d characters", maxIdempotencyKeyLength)
	}

	customerID := strings.TrimSpace(c.CustomerID)
	if customerID == "" {
		return validCreateInput{}, validationError("customerId is required")
	}

	if c.SendCurrency != CurrencyKRW {
		return validCreateInput{}, validationError("sendCurrency must be %s", CurrencyKRW)
	}
	if c.ReceiveCurrency != CurrencyPHP {
		return validCreateInput{}, validationError("receiveCurrency must be %s", CurrencyPHP)
	}

	amount, err := parseSendAmount(c.SendAmount)
	if err != nil {
		return validCreateInput{}, err
	}

	name := strings.TrimSpace(c.RecipientName)
	if name == "" {
		return validCreateInput{}, validationError("recipientName is required")
	}
	if utf8.RuneCountInString(name) > maxRecipientNameLength {
		return validCreateInput{}, validationError("recipientName must be at most %d characters", maxRecipientNameLength)
	}

	return validCreateInput{
		IdempotencyKey:  key,
		CustomerID:      customerID,
		SendCurrency:    c.SendCurrency,
		SendAmountKRW:   amount,
		ReceiveCurrency: c.ReceiveCurrency,
		RecipientName:   name,
	}, nil
}

// parseSendAmount turns the raw text into won. Only whole, positive numbers are accepted.
func parseSendAmount(raw string) (int64, error) {
	if raw == "" {
		return 0, validationError("sendAmount is required")
	}
	if decimalNumberPattern.MatchString(raw) {
		return 0, validationError("sendAmount must be a whole number without decimals")
	}
	if !wholeNumberPattern.MatchString(raw) {
		return 0, validationError("sendAmount must be a number")
	}

	amount, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || amount > MaxSendAmountKRW {
		return 0, validationError("sendAmount must be at most %d", MaxSendAmountKRW)
	}
	if amount <= 0 {
		return 0, validationError("sendAmount must be greater than 0")
	}
	if amount < MinSendAmountKRW {
		return 0, validationError("sendAmount must be at least %d", MinSendAmountKRW)
	}
	return amount, nil
}

// requestHash is a fingerprint of the business data of a create request.
// Two requests with the same Idempotency-Key must have the same hash, otherwise it is a conflict.
func (in validCreateInput) requestHash() string {
	// A struct is marshalled with a fixed field order, so equal data always gives equal bytes.
	canonical, _ := json.Marshal(struct {
		CustomerID      string `json:"customerId"`
		SendCurrency    string `json:"sendCurrency"`
		SendAmount      int64  `json:"sendAmount"`
		ReceiveCurrency string `json:"receiveCurrency"`
		RecipientName   string `json:"recipientName"`
	}{in.CustomerID, in.SendCurrency, in.SendAmountKRW, in.ReceiveCurrency, in.RecipientName})

	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}
