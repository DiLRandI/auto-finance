package parameterstore

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type ParameterStore struct {
	client *ssm.Client
}

func New(client *ssm.Client) *ParameterStore {
	return &ParameterStore{
		client: client,
	}
}

func (ps *ParameterStore) GetParameter(ctx context.Context, name string) (string, error) {
	resp, err := ps.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get parameter %s: %w", name, err)
	}

	if resp.Parameter == nil || resp.Parameter.Value == nil {
		return "", fmt.Errorf("parameter %s not found or has no value", name)
	}

	return *resp.Parameter.Value, nil
}
