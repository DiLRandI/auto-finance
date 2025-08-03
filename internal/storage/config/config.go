package config

import (
	"context"
	"fmt"

	"auto-finance/internal/storage"
)

type Client interface {
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
}

type Config struct {
	Client Client
	Bucket string
	Key    string
}

type Configuration struct {
	client Client
	Bucket string
	Key    string
}

func New(c *Config) storage.ConfigStorage {
	return &Configuration{
		client: c.Client,
		Bucket: c.Bucket,
		Key:    c.Key,
	}
}

func (c *Configuration) GetConfig(ctx context.Context) ([]byte, error) {
	data, err := c.client.GetObject(ctx, c.Bucket, c.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config from bucket %s with key %s: %w", c.Bucket, c.Key, err)
	}
	return data, nil
}
