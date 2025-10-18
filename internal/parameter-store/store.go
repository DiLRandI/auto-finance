package parameterstore

import (
	"context"
	"fmt"
	"time"

	"auto-finance/internal/errors"
	"auto-finance/internal/utils/retry"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMClient defines the interface for SSM operations used by this package
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type ParameterStore struct {
	client      SSMClient
	retryConfig retry.AWSRetryConfig
}

type Config struct {
	Client      SSMClient
	RetryConfig *retry.AWSRetryConfig
}

func New(client SSMClient) *ParameterStore {
	return &ParameterStore{
		client:      client,
		retryConfig: retry.DefaultAWSRetryConfig(),
	}
}

func NewWithConfig(config *Config) *ParameterStore {
	ps := &ParameterStore{
		client:      config.Client,
		retryConfig: retry.DefaultAWSRetryConfig(),
	}

	if config.RetryConfig != nil {
		ps.retryConfig = *config.RetryConfig
	}

	return ps
}

func (ps *ParameterStore) GetParameter(ctx context.Context, name string) (string, error) {
	var resp *ssm.GetParameterOutput
	var err error

	operation := func() error {
		resp, err = ps.client.GetParameter(ctx, &ssm.GetParameterInput{
			Name: &name,
		})
		if err != nil {
			if retry.IsAWSErrorRetryable(err) {
				return errors.NewRetryableError(
					fmt.Errorf("failed to get parameter %s: %w", name, err),
					errors.ErrorTypeAWS,
					2*time.Second,
					3,
				)
			}
			return fmt.Errorf("failed to get parameter %s: %w", name, err)
		}

		if resp.Parameter == nil || resp.Parameter.Value == nil {
			return fmt.Errorf("parameter %s not found or has no value", name)
		}

		return nil
	}

	err = retry.WithAWSRetry(ctx, ps.retryConfig, operation)
	if err != nil {
		return "", err
	}

	return *resp.Parameter.Value, nil
}

// GetParameterWithRetry allows custom retry configuration for a specific call
func (ps *ParameterStore) GetParameterWithRetry(ctx context.Context, name string, retryConfig retry.AWSRetryConfig) (string, error) {
	var resp *ssm.GetParameterOutput
	var err error

	operation := func() error {
		resp, err = ps.client.GetParameter(ctx, &ssm.GetParameterInput{
			Name: &name,
		})
		if err != nil {
			if retry.IsAWSErrorRetryable(err) {
				return errors.NewRetryableError(
					fmt.Errorf("failed to get parameter %s: %w", name, err),
					errors.ErrorTypeAWS,
					2*time.Second,
					3,
				)
			}
			return fmt.Errorf("failed to get parameter %s: %w", name, err)
		}

		if resp.Parameter == nil || resp.Parameter.Value == nil {
			return fmt.Errorf("parameter %s not found or has no value", name)
		}

		return nil
	}

	err = retry.WithAWSRetry(ctx, retryConfig, operation)
	if err != nil {
		return "", err
	}

	return *resp.Parameter.Value, nil
}
