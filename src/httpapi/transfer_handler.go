package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"finshot/src/transfer"
)

const (
	headerIdempotencyKey     = "Idempotency-Key"
	headerIdempotentReplayed = "Idempotent-Replayed"
	maxRequestBodyBytes      = 1 << 20 // 1 MB
)

// TransferService is what the handler needs from the business layer.
// Depending on this small interface (not on *transfer.Service) keeps the handler easy to test.
type TransferService interface {
	Create(ctx context.Context, cmd transfer.CreateCommand) (transfer.CreateResult, error)
	Get(ctx context.Context, id string) (transfer.Transfer, error)
	Cancel(ctx context.Context, id string) (transfer.Transfer, error)
}

// TransferHandler translates HTTP to service calls and back. It has no business rules.
type TransferHandler struct {
	service TransferService
}

func NewTransferHandler(service TransferService) *TransferHandler {
	return &TransferHandler{service: service}
}

// Create handles POST /api/transfers.
func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createTransferRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, codeValidation, "request body must be valid JSON: "+err.Error())
		return
	}

	result, err := h.service.Create(r.Context(), transfer.CreateCommand{
		IdempotencyKey:  r.Header.Get(headerIdempotencyKey),
		CustomerID:      body.CustomerID,
		SendCurrency:    body.SendCurrency,
		SendAmount:      string(body.SendAmount),
		ReceiveCurrency: body.ReceiveCurrency,
		RecipientName:   body.RecipientName,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if result.Replayed {
		w.Header().Set(headerIdempotentReplayed, "true")
	}
	writeJSON(w, http.StatusCreated, toTransferResponse(result.Transfer))
}

// Get handles GET /api/transfers/{transferId}.
func (h *TransferHandler) Get(w http.ResponseWriter, r *http.Request) {
	t, err := h.service.Get(r.Context(), r.PathValue("transferId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTransferResponse(t))
}

// Cancel handles POST /api/transfers/{transferId}/cancel.
func (h *TransferHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	t, err := h.service.Cancel(r.Context(), r.PathValue("transferId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTransferResponse(t))
}
