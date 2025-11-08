package finance

type TransactionType string

const (
	TransactionTypeCard   TransactionType = "Card"
	TransactionTypeOnline TransactionType = "Online"
	TransactionTypeATM    TransactionType = "ATM"
)

type SampathModel struct {
	TransactionType TransactionType `json:"transaction_type"`
	Identifier      string          `json:"identifier"`
	Amount          float64         `json:"amount"`
	Currency        string          `json:"currency"`
	Merchant        string          `json:"merchant"`
	Status          string          `json:"status,omitempty"`
	SmsDateTime     string          `json:"sms_date_time,omitempty"`
}
