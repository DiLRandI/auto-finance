package smsparser

type SMSParser[T any] interface {
	Parse(message string) (T, error)
}
