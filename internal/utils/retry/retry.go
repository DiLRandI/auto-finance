package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"google.golang.org/api/googleapi"
)

// AWSRetryConfig contains configuration for AWS operation retries
type AWSRetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// GoogleRetryConfig contains configuration for Google API operation retries
type GoogleRetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultAWSRetryConfig returns default AWS retry configuration
func DefaultAWSRetryConfig() AWSRetryConfig {
	return AWSRetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
	}
}

// DefaultGoogleRetryConfig returns default Google API retry configuration
func DefaultGoogleRetryConfig() GoogleRetryConfig {
	return GoogleRetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
	}
}

// IsAWSErrorRetryable checks if an AWS error should be retried
func IsAWSErrorRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for AWS specific retryable errors
	var apiErr interface {
		Error() string
		Code() string
	}

	if errors.As(err, &apiErr) {
		// Retry on throttling, rate limiting, and temporary errors
		switch apiErr.Code() {
		case "Throttling", "ThrottlingException", "RequestLimitExceeded",
			"ProvisionedThroughputExceededException", "RequestThrottled",
			"BandwidthLimitExceeded", "TooManyRequestsException",
			"ServiceUnavailable", "InternalError", "InternalFailure":
			return true
		}
	}

	// Check for connection errors and timeouts
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	return false
}

// IsGoogleErrorRetryable checks if a Google API error should be retried
func IsGoogleErrorRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for Google API specific retryable errors
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		// Retry on 5xx errors and rate limiting (429)
		if gerr.Code >= 500 && gerr.Code < 600 || gerr.Code == 429 {
			return true
		}
	}

	// Check for connection errors and timeouts
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	return false
}

// WithAWSRetry executes an AWS operation with retry logic
func WithAWSRetry(ctx context.Context, config AWSRetryConfig, operation func() error) error {
	backoff := config.InitialBackoff

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if !IsAWSErrorRetryable(err) {
			return err
		}

		if attempt < config.MaxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff = min(backoff*2, config.MaxBackoff)
			}
		}
	}

	return fmt.Errorf("aws operation failed after %d attempts", config.MaxAttempts)
}

// WithGoogleRetry executes a Google API operation with retry logic
func WithGoogleRetry(ctx context.Context, config GoogleRetryConfig, operation func() error) error {
	backoff := config.InitialBackoff

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if !IsGoogleErrorRetryable(err) {
			return err
		}

		if attempt < config.MaxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff = min(backoff*2, config.MaxBackoff)
			}
		}
	}

	return fmt.Errorf("google api operation failed after %d attempts", config.MaxAttempts)
}

// S3GetObjectWithRetry wraps S3 GetObject with retry logic
func S3GetObjectWithRetry(ctx context.Context, client *s3.Client, config AWSRetryConfig, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	var output *s3.GetObjectOutput
	var err error

	operation := func() error {
		output, err = client.GetObject(ctx, input)
		return err
	}

	err = WithAWSRetry(ctx, config, operation)
	return output, err
}

// SSMGetParameterWithRetry wraps SSM GetParameter with retry logic
func SSMGetParameterWithRetry(ctx context.Context, client *ssm.Client, config AWSRetryConfig, input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	var output *ssm.GetParameterOutput
	var err error

	operation := func() error {
		output, err = client.GetParameter(ctx, input)
		return err
	}

	err = WithAWSRetry(ctx, config, operation)
	return output, err
}

// min returns the minimum of two time.Duration values
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
