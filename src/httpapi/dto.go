package httpapi

import (
	"encoding/json"
	"time"

	"finshot/src/transfer"
)

// createTransferRequest is the JSON body of POST /api/transfers.
type createTransferRequest struct {
	CustomerID      string     `json:"customerId"`
	SendCurrency    string     `json:"sendCurrency"`
	SendAmount      amountText `json:"sendAmount"`
	ReceiveCurrency string     `json:"receiveCurrency"`
	RecipientName   string     `json:"recipientName"`
}

// amountText accepts sendAmount as a JSON string ("500000") or a JSON number (500000)
// and keeps the original text, so validation sees exactly what the client sent.
type amountText string

func (a *amountText) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*a = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*a = amountText(text)
		return nil
	}
	*a = amountText(data)
	return nil
}

// transferResponse is the JSON shape of a transfer in every response.
// All money is a string so no client ever parses it as a float.
type transferResponse struct {
	TransferID      string `json:"transferId"`
	Status          string `json:"status"`
	SendAmount      string `json:"sendAmount"`
	Fee             string `json:"fee"`
	TotalDebit      string `json:"totalDebit"`
	ReceiveAmount   string `json:"receiveAmount"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
	CustomerID      string `json:"customerId"`
	SendCurrency    string `json:"sendCurrency"`
	ReceiveCurrency string `json:"receiveCurrency"`
	FXRate          string `json:"fxRate"`
	RecipientName   string `json:"recipientName"`
}

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

func toTransferResponse(t transfer.Transfer) transferResponse {
	return transferResponse{
		TransferID:      t.ID,
		Status:          string(t.Status),
		SendAmount:      transfer.FormatKRW(t.Quote.SendAmountKRW),
		Fee:             transfer.FormatKRW(t.Quote.FeeKRW),
		TotalDebit:      transfer.FormatKRW(t.Quote.TotalDebitKRW),
		ReceiveAmount:   transfer.FormatPHP(t.Quote.ReceiveAmountCentavos),
		CreatedAt:       formatTime(t.CreatedAt),
		UpdatedAt:       formatTime(t.UpdatedAt),
		CustomerID:      t.CustomerID,
		SendCurrency:    t.SendCurrency,
		ReceiveCurrency: t.ReceiveCurrency,
		FXRate:          t.FXRate,
		RecipientName:   t.RecipientName,
	}
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

// errorResponse is the single error shape required for every non-2xx response.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
