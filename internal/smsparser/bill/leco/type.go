package leco

import (
	"time"
)

type ElectricityBill struct {
	ReadOn                time.Time `json:"read_on"`                 // Date of reading
	ImportReadingPrevious int       `json:"import_reading_previous"` // Previous import reading
	ImportReadingCurrent  int       `json:"import_reading_current"`  // Current import reading
	ImportUnits           int       `json:"import_units"`            // Import units consumed
	ExportReadingPrevious int       `json:"export_reading_previous"` // Previous export reading
	ExportReadingCurrent  int       `json:"export_reading_current"`  // Current export reading
	ExportUnits           int       `json:"export_units"`            // Export units generated
	NetUnits              int       `json:"net_units"`               // Net units (negative indicates export)
	NetUnitsType          string    `json:"net_units_type"`          // "Imp" or "Exp"
	MonthlyBill           float64   `json:"monthly_bill"`            // Monthly bill amount
	OtherCharges          float64   `json:"other_charges"`           // Additional charges
	SSCL                  float64   `json:"sscl"`                    // SSCL charges
	OpeningBalance        float64   `json:"opening_balance"`         // Balance from previous period
	OpeningBalanceDate    time.Time `json:"opening_balance_date"`    // Date of opening balance
	TotalPayable          float64   `json:"total_payable"`           // Total amount due
	LastPaymentAmount     float64   `json:"last_payment_amount"`     // Last payment amount
	LastPaymentDate       time.Time `json:"last_payment_date"`       // Date of last payment
	LastGenPayment        float64   `json:"last_gen_payment"`        // Last generation payment
}
