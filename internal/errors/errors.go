package errors

import (
	"errors"
	"fmt"
	"time"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// ErrorTypeConfig represents configuration-related errors
	ErrorTypeConfig ErrorType = "config"
	// ErrorTypeParser represents SMS parsing errors
	ErrorTypeParser ErrorType = "parser"
	// ErrorTypeAWS represents AWS service errors
	ErrorTypeAWS ErrorType = "aws"
	// ErrorTypeGoogle represents Google API errors
	ErrorTypeGoogle ErrorType = "google"
	// ErrorTypeValidation represents data validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeInternal represents internal application errors
	ErrorTypeInternal ErrorType = "internal"
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err         error
	Type        ErrorType
	RetryAfter  time.Duration
	MaxAttempts int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("%s error (retryable): %v", e.Type, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, errorType ErrorType, retryAfter time.Duration, maxAttempts int) *RetryableError {
	return &RetryableError{
		Err:         err,
		Type:        errorType,
		RetryAfter:  retryAfter,
		MaxAttempts: maxAttempts,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var retryableErr *RetryableError
	return errors.As(err, &retryableErr)
}

// GetRetryInfo returns retry information from an error
func GetRetryInfo(err error) (time.Duration, int, bool) {
	var retryableErr *RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.RetryAfter, retryableErr.MaxAttempts, true
	}
	return 0, 0, false
}

// ErrorTypeOf returns the error type of an error
func ErrorTypeOf(err error) ErrorType {
	var retryableErr *RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.Type
	}
	return ErrorTypeInternal
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(operation func() error, maxAttempts int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		if !IsRetryable(err) {
			return err
		}

		if attempt < maxAttempts {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, err)
}
