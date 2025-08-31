package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
)

func TestDefaultAWSRetryConfig(t *testing.T) {
	config := DefaultAWSRetryConfig()
	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.InitialBackoff)
	assert.Equal(t, 10*time.Second, config.MaxBackoff)
}

func TestDefaultGoogleRetryConfig(t *testing.T) {
	config := DefaultGoogleRetryConfig()
	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.InitialBackoff)
	assert.Equal(t, 10*time.Second, config.MaxBackoff)
}

func TestIsAWSErrorRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "throttling error",
			err:      &mockAWSError{code: "Throttling"},
			expected: true,
		},
		{
			name:     "request limit exceeded",
			err:      &mockAWSError{code: "RequestLimitExceeded"},
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      &mockAWSError{code: "ServiceUnavailable"},
			expected: true,
		},
		{
			name:     "internal error",
			err:      &mockAWSError{code: "InternalError"},
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      &mockAWSError{code: "AccessDenied"},
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAWSErrorRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGoogleErrorRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "500 error",
			err:      &googleapi.Error{Code: 500},
			expected: true,
		},
		{
			name:     "429 error",
			err:      &googleapi.Error{Code: 429},
			expected: true,
		},
		{
			name:     "503 error",
			err:      &googleapi.Error{Code: 503},
			expected: true,
		},
		{
			name:     "400 error",
			err:      &googleapi.Error{Code: 400},
			expected: false,
		},
		{
			name:     "404 error",
			err:      &googleapi.Error{Code: 404},
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGoogleErrorRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithAWSRetry(t *testing.T) {
	ctx := context.Background()
	config := AWSRetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		err := WithAWSRetry(ctx, config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 2 {
				return &mockAWSError{code: "Throttling"}
			}
			return nil
		}

		err := WithAWSRetry(ctx, config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		attempts := 0
		expectedErr := &mockAWSError{code: "AccessDenied"}
		operation := func() error {
			attempts++
			return expectedErr
		}

		err := WithAWSRetry(ctx, config, operation)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("exhausts all retry attempts", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return &mockAWSError{code: "Throttling"}
		}

		err := WithAWSRetry(ctx, config, operation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aws operation failed after 3 attempts")
		assert.Equal(t, 3, attempts)
	})
}

func TestWithGoogleRetry(t *testing.T) {
	ctx := context.Background()
	config := GoogleRetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		err := WithGoogleRetry(ctx, config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 2 {
				return &googleapi.Error{Code: 500}
			}
			return nil
		}

		err := WithGoogleRetry(ctx, config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		attempts := 0
		expectedErr := &googleapi.Error{Code: 400}
		operation := func() error {
			attempts++
			return expectedErr
		}

		err := WithGoogleRetry(ctx, config, operation)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("exhausts all retry attempts", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return &googleapi.Error{Code: 500}
		}

		err := WithGoogleRetry(ctx, config, operation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "google api operation failed after 3 attempts")
		assert.Equal(t, 3, attempts)
	})
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name     string
		a        time.Duration
		b        time.Duration
		expected time.Duration
	}{
		{"a < b", 1 * time.Second, 2 * time.Second, 1 * time.Second},
		{"a > b", 3 * time.Second, 2 * time.Second, 2 * time.Second},
		{"a == b", 5 * time.Second, 5 * time.Second, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock AWS error for testing
type mockAWSError struct {
	code string
}

func (m *mockAWSError) Error() string {
	return "mock AWS error: " + m.code
}

func (m *mockAWSError) Code() string {
	return m.code
}

// Test context cancellation
func TestWithAWSRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := AWSRetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	cancel() // Cancel context immediately

	operation := func() error {
		return &mockAWSError{code: "Throttling"}
	}

	err := WithAWSRetry(ctx, config, operation)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWithGoogleRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := GoogleRetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	cancel() // Cancel context immediately

	operation := func() error {
		return &googleapi.Error{Code: 500}
	}

	err := WithGoogleRetry(ctx, config, operation)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
