package transfer

import "fmt"

// All money is handled as int64 in the smallest unit of its currency:
//   - KRW in won (KRW has no minor unit)
//   - PHP in centavos (1 PHP = 100 centavos)
//
// Floats are never used, so every amount is exact.
const (
	CurrencyKRW = "KRW"
	CurrencyPHP = "PHP"

	MinSendAmountKRW = 10_000
	// MaxSendAmountKRW keeps sendAmount × 412 far below the int64 limit.
	MaxSendAmountKRW = 1_000_000_000_000

	minFeeKRW  = 3_000
	feePercent = 1

	// FX rate 1 KRW = 0.0412 PHP, written as the fraction 412 / 10_000.
	FXRate            = "0.0412"
	fxRateNumerator   = 412
	fxRateDenominator = 10_000

	centavosPerPeso = 100
)

// Quote is the price of a transfer, calculated once when the transfer is created.
type Quote struct {
	SendAmountKRW         int64
	FeeKRW                int64
	TotalDebitKRW         int64
	ReceiveAmountCentavos int64
}

// NewQuote applies the pricing rules to a validated send amount.
// Rounding happens only here, HALF_UP, and the result is stored as is.
func NewQuote(sendAmountKRW int64) Quote {
	fee := max(minFeeKRW, divideRoundHalfUp(sendAmountKRW*feePercent, 100))

	// sendAmount × 412 / 10_000 gives pesos; × 100 gives centavos.
	// The two steps combine into a single division by 100, so we round only once.
	receiveCentavos := divideRoundHalfUp(
		sendAmountKRW*fxRateNumerator*centavosPerPeso,
		fxRateDenominator,
	)

	return Quote{
		SendAmountKRW:         sendAmountKRW,
		FeeKRW:                fee,
		TotalDebitKRW:         sendAmountKRW + fee,
		ReceiveAmountCentavos: receiveCentavos,
	}
}

// divideRoundHalfUp divides two non-negative integers and rounds half up.
// Example: 300_040 / 100 = 3000.4 -> 3000, and 300_050 / 100 = 3000.5 -> 3001.
func divideRoundHalfUp(numerator, denominator int64) int64 {
	return (numerator + denominator/2) / denominator
}

// FormatKRW renders a won amount for the API, e.g. 500000 -> "500000".
func FormatKRW(won int64) string {
	return fmt.Sprintf("%d", won)
}

// FormatPHP renders centavos with exactly two decimals, e.g. 2060000 -> "20600.00".
func FormatPHP(centavos int64) string {
	return fmt.Sprintf("%d.%02d", centavos/centavosPerPeso, centavos%centavosPerPeso)
}
