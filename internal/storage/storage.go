package storage

import (
	"context"
)

type MessageStorage[T any] interface {
	Save(ctx context.Context, message T) error
	// Read(ctx context.Context, id uuid.UUID) (T, error)
	// ReadAll(ctx context.Context, pageSize, pageNumber int) ([]T, error)
	// Delete(ctx context.Context, id uuid.UUID) error
}
