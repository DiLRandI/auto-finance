package smsparser

type SMSParser[T any] interface {
	GetName() string
	Parse(message string) (T, error)
}

type UniversalParser interface {
	GetName() string
	Parse(message string) (interface{}, error) // Note: returns interface{}
}

type GenericSMSParseWrapper[T any] struct {
	parser SMSParser[T]
}

func (g GenericSMSParseWrapper[T]) GetName() string {
	return g.parser.GetName()
}

func (g GenericSMSParseWrapper[T]) Parse(message string) (interface{}, error) {
	return g.parser.Parse(message)
}

// NewGenericParserWrapper creates a new wrapper for a generic parser
func NewGenericParserWrapper[T any](p SMSParser[T]) UniversalParser {
	return GenericSMSParseWrapper[T]{parser: p}
}
