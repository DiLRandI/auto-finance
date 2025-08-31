package finance

type TransactionType string

const (
	TransactionTypeCard   TransactionType = "Card"
	TransactionTypeOnline TransactionType = "Online"
	TransactionTypeATM    TransactionType = "ATM"
)

type SampathModel struct {
	TransactionType TransactionType `json:"transaction_type"`
	Amount          float64         `json:"amount"`
	Currency        string          `json:"currency"`
	ReferenceNo     string          `json:"reference_no"`
	Merchant        string          `json:"merchant"`
}
