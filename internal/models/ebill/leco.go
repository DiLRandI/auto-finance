package ebill

import "time"

type ElectricityBill struct {
	AccountNumber      string    `json:"accountNumber"`
	AccountType        string    `json:"accountType"`
	AccountName        string    `json:"accountName"`
	ReadOn             time.Time `json:"readOn"`
	ImportPrevious     int       `json:"importPrevious"`
	ImportCurrent      int       `json:"importCurrent"`
	ImportUnits        int       `json:"importUnits"`
	ExportPrevious     int       `json:"exportPrevious"`
	ExportCurrent      int       `json:"exportCurrent"`
	ExportUnits        int       `json:"exportUnits"`
	NetUnits           int       `json:"netUnits"`
	NetUnitsType       string    `json:"netUnitsType"`
	MonthlyBill        float64   `json:"monthlyBill"`
	OtherCharges       float64   `json:"otherCharges"`
	SSCL               float64   `json:"sscl"`
	OpeningBalance     float64   `json:"openingBalance"`
	OpeningBalanceDate time.Time `json:"openingBalanceDate"`
	TotalPayable       float64   `json:"totalPayable"`
	LastPaymentAmount  float64   `json:"lastPaymentAmount"`
	LastPaymentDate    time.Time `json:"lastPaymentDate"`
	LastGenPayment     float64   `json:"lastGenPayment"`
}
