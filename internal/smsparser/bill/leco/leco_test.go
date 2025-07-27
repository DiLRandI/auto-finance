package leco

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLecoParser_Parse(t *testing.T) {
	sms := `Read On: 27-JUL-25
Imp: 12345-12367=22
Exp: 54321-54330=9
Net Units: 13 (Imp)
Monthly Bill: Rs. 1,234.56
Other Charges: Rs. 78.90
SSCL: Rs. 12.34
Opening Balance: Rs. 100.00 on 01-JUL-25
Total Payable: Rs. 1,425.80
Last Payment: Rs. 1,000.00 on 15-JUL-25
Last Amount Paid for Generation: Rs. 50.00`

	parser := &LecoParser{}
	bill, err := parser.Parse(sms)
	assert.NoError(t, err)

	readOn, _ := time.Parse("02-Jan-06", "27-Jul-25")
	openingBalanceDate, _ := time.Parse("02-Jan-06", "01-Jul-25")
	lastPaymentDate, _ := time.Parse("02-Jan-06", "15-Jul-25")

	expected := &ElectricityBill{
		ReadOn:                readOn,
		ImportReadingPrevious: 12345,
		ImportReadingCurrent:  12367,
		ImportUnits:           22,
		ExportReadingPrevious: 54321,
		ExportReadingCurrent:  54330,
		ExportUnits:           9,
		NetUnits:              13,
		NetUnitsType:          "Imp",
		MonthlyBill:           1234.56,
		OtherCharges:          78.90,
		SSCL:                  12.34,
		OpeningBalance:        100.00,
		OpeningBalanceDate:    openingBalanceDate,
		TotalPayable:          1425.80,
		LastPaymentAmount:     1000.00,
		LastPaymentDate:       lastPaymentDate,
		LastGenPayment:        50.00,
	}

	assert.Equal(t, expected, bill)
}
