package transfer

import "testing"

func TestNewQuote(t *testing.T) {
	tests := []struct {
		name        string
		sendAmount  int64
		wantFee     string
		wantDebit   string
		wantReceive string
	}{
		{"example from the assignment", 500_000, "5000", "505000", "20600.00"},
		{"minimum amount pays the minimum fee", 10_000, "3000", "13000", "412.00"},
		{"1% is exactly the minimum fee", 300_000, "3000", "303000", "12360.00"},
		{"fee 3000.5 rounds half up to 3001", 300_050, "3001", "303051", "12362.06"},
		{"fee 3000.4 rounds down, receive 12361.648 rounds up", 300_040, "3000", "303040", "12361.65"},
		{"large amount stays exact", 999_999_999_999, "10000000000", "1009999999999", "41199999999.96"},
		{"receive 508.614 rounds down to 508.61", 12_345, "3000", "15345", "508.61"},
		{"3000000 gives exactly 123600.00", 3_000_000, "30000", "3030000", "123600.00"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQuote(tc.sendAmount)

			if got := FormatKRW(q.FeeKRW); got != tc.wantFee {
				t.Errorf("fee = %s, want %s", got, tc.wantFee)
			}
			if got := FormatKRW(q.TotalDebitKRW); got != tc.wantDebit {
				t.Errorf("totalDebit = %s, want %s", got, tc.wantDebit)
			}
			if got := FormatPHP(q.ReceiveAmountCentavos); got != tc.wantReceive {
				t.Errorf("receiveAmount = %s, want %s", got, tc.wantReceive)
			}
		})
	}
}
