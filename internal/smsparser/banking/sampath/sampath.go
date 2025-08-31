package sampath

import (
	"auto-finance/internal/models/finance"
	"auto-finance/internal/smsparser"
	"fmt"
)

type parser struct{}

func New() smsparser.SMSParser[*finance.SampathModel] {
	return &parser{}
}

func (p *parser) GetName() string {
	return "Sampath Bank Parser"
}
func (p *parser) Parse(sms string) (*finance.SampathModel, error) {
	fmt.Println("===============================================================================")
	fmt.Printf("%+v\n", sms)
	fmt.Println("===============================================================================")

	return &finance.SampathModel{}, nil
}
