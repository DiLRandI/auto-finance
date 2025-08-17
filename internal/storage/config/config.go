package config

import (
	"context"
	"fmt"
	"io"

	"auto-finance/internal/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type Config struct {
	Client Client
	Bucket string
}

type Configuration struct {
	client Client
	Bucket string
}

func New(c *Config) storage.ConfigStorage {
	return &Configuration{
		client: c.Client,
		Bucket: c.Bucket,
	}
}

func (c *Configuration) GetConfig(ctx context.Context, key string) ([]byte, error) {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get config from bucket %s with key %s: %w", c.Bucket, key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from bucket %s with key %s: %w", c.Bucket, key, err)
	}

	return data, nil
}
