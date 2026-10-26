package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"finshot/src/transfer"
)

const (
	codeValidation = "VALIDATION_ERROR"
	codeNotFound   = "NOT_FOUND"
	codeConflict   = "CONFLICT"
	codeInternal   = "INTERNAL_ERROR"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: code, Message: message})
}

// writeServiceError maps a business error kind to its HTTP status.
// Anything unexpected becomes a 500 and is logged, without leaking details to the client.
func writeServiceError(w http.ResponseWriter, err error) {
	var businessErr *transfer.Error
	if !errors.As(err, &businessErr) {
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, codeInternal, "unexpected server error")
		return
	}

	switch businessErr.Kind {
	case transfer.KindValidation:
		writeError(w, http.StatusBadRequest, codeValidation, businessErr.Message)
	case transfer.KindNotFound:
		writeError(w, http.StatusNotFound, codeNotFound, businessErr.Message)
	case transfer.KindConflict:
		writeError(w, http.StatusConflict, codeConflict, businessErr.Message)
	default:
		log.Printf("unmapped error kind %q: %v", businessErr.Kind, err)
		writeError(w, http.StatusInternalServerError, codeInternal, "unexpected server error")
	}
}
