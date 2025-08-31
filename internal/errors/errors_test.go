package errors

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRetryableError(t *testing.T) {
	originalErr := errors.New("original error")
	retryableErr := NewRetryableError(originalErr, ErrorTypeAWS, 2*time.Second, 3)

	assert.NotNil(t, retryableErr)
	assert.Equal(t, ErrorTypeAWS, retryableErr.Type)
	assert.Equal(t, 2*time.Second, retryableErr.RetryAfter)
	assert.Equal(t, 3, retryableErr.MaxAttempts)
	assert.Equal(t, originalErr, retryableErr.Err)
}

func TestRetryableError_Error(t *testing.T) {
	originalErr := errors.New("connection timeout")
	retryableErr := NewRetryableError(originalErr, ErrorTypeAWS, 2*time.Second, 3)

	expected := "aws error (retryable): connection timeout"
	assert.Equal(t, expected, retryableErr.Error())
}

func TestRetryableError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	retryableErr := NewRetryableError(originalErr, ErrorTypeAWS, 2*time.Second, 3)

	unwrapped := errors.Unwrap(retryableErr)
	assert.Equal(t, originalErr, unwrapped)
}

func TestIsRetryable(t *testing.T) {
	t.Run("with retryable error", func(t *testing.T) {
		originalErr := errors.New("retryable error")
		retryableErr := NewRetryableError(originalErr, ErrorTypeAWS, 2*time.Second, 3)

		assert.True(t, IsRetryable(retryableErr))
	})

	t.Run("with non-retryable error", func(t *testing.T) {
		nonRetryableErr := errors.New("non-retryable error")
		assert.False(t, IsRetryable(nonRetryableErr))
	})

	t.Run("with nil error", func(t *testing.T) {
		assert.False(t, IsRetryable(nil))
	})
}

func TestGetRetryInfo(t *testing.T) {
	t.Run("with retryable error", func(t *testing.T) {
		originalErr := errors.New("retryable error")
		retryableErr := NewRetryableError(originalErr, ErrorTypeGoogle, 5*time.Second, 5)

		retryAfter, maxAttempts, found := GetRetryInfo(retryableErr)
		assert.True(t, found)
		assert.Equal(t, 5*time.Second, retryAfter)
		assert.Equal(t, 5, maxAttempts)
	})

	t.Run("with non-retryable error", func(t *testing.T) {
		nonRetryableErr := errors.New("non-retryable error")

		retryAfter, maxAttempts, found := GetRetryInfo(nonRetryableErr)
		assert.False(t, found)
		assert.Equal(t, time.Duration(0), retryAfter)
		assert.Equal(t, 0, maxAttempts)
	})
}

func TestErrorTypeOf(t *testing.T) {
	t.Run("with retryable error", func(t *testing.T) {
		originalErr := errors.New("some error")
		retryableErr := NewRetryableError(originalErr, ErrorTypeConfig, 1*time.Second, 2)

		errorType := ErrorTypeOf(retryableErr)
		assert.Equal(t, ErrorTypeConfig, errorType)
	})

	t.Run("with non-retryable error", func(t *testing.T) {
		nonRetryableErr := errors.New("non-retryable error")

		errorType := ErrorTypeOf(nonRetryableErr)
		assert.Equal(t, ErrorTypeInternal, errorType)
	})

	t.Run("with nil error", func(t *testing.T) {
		errorType := ErrorTypeOf(nil)
		assert.Equal(t, ErrorTypeInternal, errorType)
	})
}

func TestRetryWithBackoff(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		err := RetryWithBackoff(operation, 3, 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 2 {
				return NewRetryableError(errors.New("temporary error"), ErrorTypeAWS, 100*time.Millisecond, 3)
			}
			return nil
		}

		err := RetryWithBackoff(operation, 3, 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		attempts := 0
		expectedErr := errors.New("permanent error")
		operation := func() error {
			attempts++
			return expectedErr
		}

		err := RetryWithBackoff(operation, 3, 100*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("exhausts all retry attempts", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return NewRetryableError(errors.New("always failing"), ErrorTypeAWS, 100*time.Millisecond, 3)
		}

		err := RetryWithBackoff(operation, 3, 100*time.Millisecond)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operation failed after 3 attempts")
		assert.Equal(t, 3, attempts)
	})

	t.Run("with mixed retryable and non-retryable errors", func(t *testing.T) {
		attempts := 0
		expectedErr := errors.New("permanent error")
		operation := func() error {
			attempts++
			if attempts == 1 {
				return NewRetryableError(errors.New("temporary error"), ErrorTypeAWS, 100*time.Millisecond, 3)
			}
			return expectedErr
		}

		err := RetryWithBackoff(operation, 3, 100*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 2, attempts)
	})
}

func TestErrorTypeConstants(t *testing.T) {
	testCases := []struct {
		name      string
		errorType ErrorType
		expected  string
	}{
		{"Config", ErrorTypeConfig, "config"},
		{"Parser", ErrorTypeParser, "parser"},
		{"AWS", ErrorTypeAWS, "aws"},
		{"Google", ErrorTypeGoogle, "google"},
		{"Validation", ErrorTypeValidation, "validation"},
		{"Internal", ErrorTypeInternal, "internal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.errorType))
		})
	}
}
