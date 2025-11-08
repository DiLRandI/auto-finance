package sampath_test

import (
	"testing"

	"auto-finance/internal/models/finance"
	"auto-finance/internal/smsparser/banking/sampath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseSamples(t *testing.T) {
	t.Parallel()

	parser := sampath.New()

	tests := []struct {
		name string
		sms  string
		want *finance.SampathModel
	}{
		{
			name: "card authorization with commas and newlines",
			sms: `Cr Crd no..**1234 Auth Pmt LKR 6,789.50 at MASKED BISTRO (PVT) LTD Avl Bal LKR 40,289.06 Enq Call 0112000000
Sampath Bank 07-NOV`,
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "1234",
				Amount:          6789.50,
				Currency:        "LKR",
				Merchant:        "MASKED BISTRO (PVT) LTD",
				Status:          "authorized",
			},
		},
		{
			name: "card authorization with hyphenated merchant",
			sms:  "Cr Crd no..**5678 Auth Pmt LKR 1,250.00 at MASKED-PARK TERMINAL Avl Bal LKR 41,648.06 Enq Call 0112000000 Sampath Bank 07-NOV",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "5678",
				Amount:          1250.00,
				Currency:        "LKR",
				Merchant:        "MASKED-PARK TERMINAL",
				Status:          "authorized",
			},
		},
		{
			name: "card authorization with tilde separated merchant",
			sms:  "Cr Crd no..**9999 Auth Pmt LKR 7,624.00 at MASKED~PETROLEUM~PLC Avl Bal LKR 39,024.06 Enq Call 0112000000 Sampath Bank 07-NOV",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "9999",
				Amount:          7624.00,
				Currency:        "LKR",
				Merchant:        "MASKED PETROLEUM PLC",
				Status:          "authorized",
			},
		},
		{
			name: "card reversal in usd",
			sms:  "Cr Crd no..**1111 Rvsd Pmt USD 1.00 at MASKED TEMP HOLD Avl Bal LKR 421,956.98 Enq Call 0112000000 Sampath Bank 06-NOV",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "1111",
				Amount:          1.00,
				Currency:        "USD",
				Merchant:        "MASKED TEMP HOLD",
				Status:          "reversed",
			},
		},
		{
			name: "card authorization usd zero amount",
			sms:  "Cr Crd no..**2222 Auth Pmt USD .00 at MASKED ZERO TEST Avl Bal LKR 440,425.09 Enq Call 0112000000 Sampath Bank 03-NOV",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "2222",
				Amount:          0.00,
				Currency:        "USD",
				Merchant:        "MASKED ZERO TEST",
				Status:          "authorized",
			},
		},
		{
			name: "web channel authorization",
			sms:  "Web Crd no..**2862 Auth Pmt USD 1.03 at MASKED WEB SERVICE Avl Bal LKR 19,051.77 Enq Call 0112000000 Sampath Bank 02-NOV",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "2862",
				Amount:          1.03,
				Currency:        "USD",
				Merchant:        "MASKED WEB SERVICE",
				Status:          "authorized",
			},
		},
		{
			name: "card payment credited",
			sms:  "Cr Crd no..**3333 Credited LKR 50,000.00 for MASKED PAYMENT RECEIVED - CLIENT Avl Bal LKR 504,024.06 Enq Call 0112000000 Sampath Bank 08-NOV",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeCard,
				Identifier:      "3333",
				Amount:          50000.00,
				Currency:        "LKR",
				Merchant:        "MASKED PAYMENT RECEIVED - CLIENT",
				Status:          "credit",
			},
		},
		{
			name: "atm cash withdrawal",
			sms:  "LKR 4,005.00 debited from AC **4060 via ATM at MASKED BANK ATM CITY For Inq Call 0112000000, Sampath Bank",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeATM,
				Identifier:      "4060",
				Amount:          4005.00,
				Currency:        "LKR",
				Merchant:        "MASKED BANK ATM CITY",
				Status:          "debit",
			},
		},
		{
			name: "account credit transfer",
			sms:  "LKR 4,280.00 credited to AC **4060 for MASKED INSTALLMENT TRANSFER 8 of 12 For Inq Call 0112000000, Sampath Bank",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeOnline,
				Identifier:      "4060",
				Amount:          4280.00,
				Currency:        "LKR",
				Merchant:        "MASKED INSTALLMENT TRANSFER 8 of 12",
				Status:          "credit",
			},
		},
		{
			name: "account debit card payment",
			sms:  "LKR 200,000.00 debited from AC **0004 for MASKED CARD PAYMENT 123456789 For Inq Call 0112000000, Sampath Bank",
			want: &finance.SampathModel{
				TransactionType: finance.TransactionTypeOnline,
				Identifier:      "0004",
				Amount:          200000.00,
				Currency:        "LKR",
				Merchant:        "MASKED CARD PAYMENT 123456789",
				Status:          "debit",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parser.Parse(tt.sms)
			require.NoError(t, err)
			require.NotEmpty(t, got.SmsDateTime, "parser should stamp the SMS time")
			got.SmsDateTime = ""
			assert.Equal(t, tt.want, got)
		})
	}
}
