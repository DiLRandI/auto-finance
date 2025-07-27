package storage

import (
	"context"

	"auto-finance/internal/models"

	"github.com/google/uuid"
)

type MessageStorage interface {
	Save(ctx context.Context, message *models.Message) error
	Read(ctx context.Context, id uuid.UUID) (*models.Message, error)
	ReadAll(ctx context.Context, pageSize, pageNumber int) ([]*models.Message, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
