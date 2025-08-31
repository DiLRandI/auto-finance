package parameterstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"auto-finance/internal/utils/retry"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSSMClient is a mock implementation of the SSM client for testing
type MockSSMClient struct {
	mock.Mock
}

func (m *MockSSMClient) GetParameter(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ssm.GetParameterOutput), args.Error(1)
}

func TestParameterStore_GetParameter(t *testing.T) {
	t.Run("successful parameter retrieval", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		parameterName := "/test/parameter"
		parameterValue := "test-value"

		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(&ssm.GetParameterOutput{
			Parameter: &types.Parameter{
				Value: aws.String(parameterValue),
			},
		}, nil)

		store := New(mockClient)
		result, err := store.GetParameter(context.Background(), parameterName)

		assert.NoError(t, err)
		assert.Equal(t, parameterValue, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("retry on transient AWS error", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		parameterName := "/test/parameter"
		parameterValue := "test-value"

		// First call fails with throttling error (retryable), second succeeds
		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(nil, &mockAWSError{code: "Throttling"}).Once()

		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(&ssm.GetParameterOutput{
			Parameter: &types.Parameter{
				Value: aws.String(parameterValue),
			},
		}, nil).Once()

		store := NewWithConfig(&Config{
			Client: mockClient,
			RetryConfig: &retry.AWSRetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 10 * time.Millisecond,
				MaxBackoff:     100 * time.Millisecond,
			},
		})

		result, err := store.GetParameter(context.Background(), parameterName)

		assert.NoError(t, err)
		assert.Equal(t, parameterValue, result)
		mockClient.AssertNumberOfCalls(t, "GetParameter", 2)
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		parameterName := "/test/parameter"
		expectedErr := errors.New("access denied")

		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(nil, expectedErr).Once()

		store := New(mockClient)
		result, err := store.GetParameter(context.Background(), parameterName)

		assert.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), "access denied")
		mockClient.AssertNumberOfCalls(t, "GetParameter", 1)
	})

	t.Run("parameter not found error", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		parameterName := "/nonexistent/parameter"

		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(&ssm.GetParameterOutput{
			Parameter: nil,
		}, nil).Once()

		store := New(mockClient)
		result, err := store.GetParameter(context.Background(), parameterName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found or has no value")
		assert.Equal(t, "", result)
		mockClient.AssertExpectations(t)
	})

	t.Run("exhausts all retry attempts", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		parameterName := "/test/parameter"

		// All calls fail with throttling errors
		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(nil, &mockAWSError{code: "Throttling"}).Times(3)

		store := NewWithConfig(&Config{
			Client: mockClient,
			RetryConfig: &retry.AWSRetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 10 * time.Millisecond,
				MaxBackoff:     100 * time.Millisecond,
			},
		})

		result, err := store.GetParameter(context.Background(), parameterName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aws operation failed after 3 attempts")
		assert.Equal(t, "", result)
		mockClient.AssertNumberOfCalls(t, "GetParameter", 3)
	})
}

func TestParameterStore_GetParameterWithRetry(t *testing.T) {
	t.Run("custom retry configuration", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		parameterName := "/test/parameter"
		parameterValue := "test-value"

		// First call fails, second succeeds
		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(nil, &mockAWSError{code: "Throttling"}).Once()

		mockClient.On("GetParameter", mock.Anything, &ssm.GetParameterInput{
			Name: aws.String(parameterName),
		}).Return(&ssm.GetParameterOutput{
			Parameter: &types.Parameter{
				Value: aws.String(parameterValue),
			},
		}, nil).Once()

		store := New(mockClient)
		customRetryConfig := retry.AWSRetryConfig{
			MaxAttempts:    2,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     100 * time.Millisecond,
		}

		result, err := store.GetParameterWithRetry(context.Background(), parameterName, customRetryConfig)

		assert.NoError(t, err)
		assert.Equal(t, parameterValue, result)
		mockClient.AssertNumberOfCalls(t, "GetParameter", 2)
	})
}

func TestNewParameterStore(t *testing.T) {
	t.Run("default constructor", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		store := New(mockClient)

		assert.NotNil(t, store)
		assert.Equal(t, mockClient, store.client)
	})

	t.Run("constructor with config", func(t *testing.T) {
		mockClient := new(MockSSMClient)
		customRetryConfig := retry.AWSRetryConfig{
			MaxAttempts:    5,
			InitialBackoff: 2 * time.Second,
			MaxBackoff:     10 * time.Second,
		}

		store := NewWithConfig(&Config{
			Client:      mockClient,
			RetryConfig: &customRetryConfig,
		})

		assert.NotNil(t, store)
		assert.Equal(t, mockClient, store.client)
		assert.Equal(t, customRetryConfig, store.retryConfig)
	})

	t.Run("constructor with nil retry config", func(t *testing.T) {
		mockClient := new(MockSSMClient)

		store := NewWithConfig(&Config{
			Client:      mockClient,
			RetryConfig: nil,
		})

		assert.NotNil(t, store)
		assert.Equal(t, mockClient, store.client)
		assert.Equal(t, retry.DefaultAWSRetryConfig(), store.retryConfig)
	})
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
