package autofinance

type Request struct {
	Sender string `json:"sender"`
	Body   string `json:"body"`
	Test   bool   `json:"test"`
}
