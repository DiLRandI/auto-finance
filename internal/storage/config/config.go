package config

import (
	"context"
	"fmt"
	"io"
	"time"

	"auto-finance/internal/errors"
	"auto-finance/internal/utils/retry"

	"auto-finance/internal/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Enhanced configuration storage with retry capabilities
type Config struct {
	Client      Client
	Bucket      string
	RetryConfig *retry.AWSRetryConfig
}

type EnhancedConfiguration struct {
	client      Client
	Bucket      string
	retryConfig retry.AWSRetryConfig
}

func New(c *Config) storage.ConfigStorage {
	retryConfig := retry.DefaultAWSRetryConfig()
	if c.RetryConfig != nil {
		retryConfig = *c.RetryConfig
	}

	return &EnhancedConfiguration{
		client:      c.Client,
		Bucket:      c.Bucket,
		retryConfig: retryConfig,
	}
}

func (ec *EnhancedConfiguration) GetConfig(ctx context.Context, key string) ([]byte, error) {
	var out *s3.GetObjectOutput
	var err error

	operation := func() error {
		out, err = ec.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(ec.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to get config from bucket %s with key %s: %w", ec.Bucket, key, err),
				errors.ErrorTypeAWS,
				2*time.Second,
				3,
			)
		}
		return nil
	}

	err = retry.WithAWSRetry(ctx, ec.retryConfig, operation)
	if err != nil {
		return nil, err
	}

	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from bucket %s with key %s: %w", ec.Bucket, key, err)
	}

	return data, nil
}
