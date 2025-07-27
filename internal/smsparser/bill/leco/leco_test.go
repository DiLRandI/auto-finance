package leco_test

import (
	"auto-finance/internal/smsparser/bill/leco"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	parser := leco.New()
	assert.NotNil(t, parser, "New() should return a non-nil parser")
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		sms     string
		want    *leco.ElectricityBill
		wantErr bool
	}{
		{
			name: "valid complete SMS with all fields",
			sms: `A/N: 123456789 (Domestic)
Account Name Example
Read On: 27-JUL-25
Imp: 12345-12367=22
Exp: 54321-54330=9
Net Units: 13 (Imp)
Monthly Bill: Rs. 1,234.56
Other Charges: Rs. 78.90
SSCL: Rs. 12.34
Opening Balance: Rs. 100.00 on 01-JUL-25
Total Payable: Rs. 1,425.80
Last Payment: Rs. 1,000.00 on 15-JUL-25
Last Amount Paid for Generation: Rs. 50.00`,
			want: func() *leco.ElectricityBill {
				readOn, _ := time.Parse("02-Jan-06", "27-Jul-25")
				openingBalanceDate, _ := time.Parse("02-Jan-06", "01-Jul-25")
				lastPaymentDate, _ := time.Parse("02-Jan-06", "15-Jul-25")
				return &leco.ElectricityBill{
					AccountNumber:      "123456789",
					AccountType:        "Domestic",
					AccountName:        "Account Name Example",
					ReadOn:             readOn,
					ImportPrevious:     12345,
					ImportCurrent:      12367,
					ImportUnits:        22,
					ExportPrevious:     54321,
					ExportCurrent:      54330,
					ExportUnits:        9,
					NetUnits:           13,
					NetUnitsType:       "Imp",
					MonthlyBill:        1234.56,
					OtherCharges:       78.90,
					SSCL:               12.34,
					OpeningBalance:     100.00,
					OpeningBalanceDate: openingBalanceDate,
					TotalPayable:       1425.80,
					LastPaymentAmount:  1000.00,
					LastPaymentDate:    lastPaymentDate,
					LastGenPayment:     50.00,
				}
			}(),
			wantErr: false,
		},
		{
			name: "valid SMS with account number only",
			sms: `A/N: 987654321
Read On: 15-DEC-24
Imp: 5000-5100=100`,
			want: func() *leco.ElectricityBill {
				readOn, _ := time.Parse("02-Jan-06", "15-Dec-24")
				return &leco.ElectricityBill{
					AccountNumber:  "987654321",
					ReadOn:         readOn,
					ImportPrevious: 5000,
					ImportCurrent:  5100,
					ImportUnits:    100,
				}
			}(),
			wantErr: false,
		},
		{
			name:    "invalid date format - wrong separator",
			sms:     "Read On: 2025/07/27",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name:    "invalid date format - incomplete",
			sms:     "Read On: 27-JUL",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name:    "invalid reading format - missing parts",
			sms:     "Imp: 12345-12367",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name:    "invalid reading format - non-numeric",
			sms:     "Imp: abc-def=ghi",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name:    "invalid amount format - no numeric value",
			sms:     "Monthly Bill: Rs. abc",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name:    "invalid net units format",
			sms:     "Net Units: abc (Imp)",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name:    "invalid account format - no number",
			sms:     "A/N: (Domestic)",
			want:    &leco.ElectricityBill{},
			wantErr: true,
		},
		{
			name: "SMS with prefixed lines (>)",
			sms: `>A/N: 111222333
>Read On: 01-JAN-25
>Imp: 1000-1050=50`,
			want: func() *leco.ElectricityBill {
				readOn, _ := time.Parse("02-Jan-06", "01-Jan-25")
				return &leco.ElectricityBill{
					AccountNumber:  "111222333",
					ReadOn:         readOn,
					ImportPrevious: 1000,
					ImportCurrent:  1050,
					ImportUnits:    50,
				}
			}(),
			wantErr: false,
		},
		{
			name: "negative net units (export)",
			sms: `Net Units: -25 (Exp)
Monthly Bill: Rs. 500.00`,
			want: &leco.ElectricityBill{
				NetUnits:     -25,
				NetUnitsType: "Exp",
				MonthlyBill:  500.00,
			},
			wantErr: false,
		},
		{
			name: "amount with commas",
			sms: `Monthly Bill: Rs. 12,345.67
Total Payable: Rs. 15,000.00`,
			want: &leco.ElectricityBill{
				MonthlyBill:  12345.67,
				TotalPayable: 15000.00,
			},
			wantErr: false,
		},
		{
			name:    "empty SMS",
			sms:     "",
			want:    &leco.ElectricityBill{},
			wantErr: false, // Empty SMS should not error, just return empty bill
		},
		{
			name: "SMS with empty lines and whitespace",
			sms: `

Read On: 01-JAN-25

Imp: 1000-1050=50

`,
			want: func() *leco.ElectricityBill {
				readOn, _ := time.Parse("02-Jan-06", "01-Jan-25")
				return &leco.ElectricityBill{
					ReadOn:         readOn,
					ImportPrevious: 1000,
					ImportCurrent:  1050,
					ImportUnits:    50,
				}
			}(),
			wantErr: false,
		},
		{
			name: "incomplete account name handling",
			sms:  "A/N: 123456789 (Domestic)",
			want: &leco.ElectricityBill{
				AccountNumber: "123456789",
				AccountType:   "Domestic",
			},
			wantErr: true, // Should error because account name is expected but missing
		},
		{
			name: "balance without date",
			sms:  "Opening Balance: Rs. 500.00",
			want: &leco.ElectricityBill{
				OpeningBalance: 500.00,
			},
			wantErr: false,
		},
		{
			name: "payment without date",
			sms:  "Last Payment: Rs. 1000.00",
			want: &leco.ElectricityBill{
				LastPaymentAmount: 1000.00,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := leco.New()
			got, err := parser.Parse(tt.sms)

			if tt.wantErr {
				assert.Error(t, err, "expected an error but got none for SMS: %s", tt.sms)
			} else {
				assert.NoError(t, err, "unexpected error: %v for SMS: %s", err, tt.sms)
				assert.Equal(t, tt.want, got, "parsed bill does not match expected for SMS: %s", tt.sms)
			}
		})
	}
}

// Test specific error conditions to ensure proper error handling
func TestParser_Parse_ErrorHandling(t *testing.T) {
	parser := leco.New()

	// Test that first error is returned, not subsequent ones
	sms := `Read On: invalid-date
Imp: invalid-reading
Monthly Bill: invalid-amount`

	_, err := parser.Parse(sms)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Read On", "Should contain the field name that caused the first error")
}

// Test edge cases for field parsing
func TestParser_Parse_EdgeCases(t *testing.T) {
	parser := leco.New()

	tests := []struct {
		name  string
		sms   string
		check func(t *testing.T, bill *leco.ElectricityBill, err error)
	}{
		{
			name: "export units higher than import",
			sms:  "Exp: 1000-2000=1000",
			check: func(t *testing.T, bill *leco.ElectricityBill, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1000, bill.ExportPrevious)
				assert.Equal(t, 2000, bill.ExportCurrent)
				assert.Equal(t, 1000, bill.ExportUnits)
			},
		},
		{
			name: "zero amounts",
			sms:  "Monthly Bill: Rs. 0.00",
			check: func(t *testing.T, bill *leco.ElectricityBill, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 0.0, bill.MonthlyBill)
			},
		},
		{
			name: "account number without type",
			sms:  "A/N: 123456789",
			check: func(t *testing.T, bill *leco.ElectricityBill, err error) {
				assert.Error(t, err) // pendingAcctName should cause error
				assert.Equal(t, "123456789", bill.AccountNumber)
				assert.Empty(t, bill.AccountType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.sms)
			tt.check(t, got, err)
		})
	}
}
