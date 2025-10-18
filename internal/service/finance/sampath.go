package finance

import (
	"context"

	"auto-finance/internal/models/finance"
	"auto-finance/internal/storage"

	"github.com/rs/zerolog"
)

type SampathBillService interface {
	HandleSampathBill(ctx context.Context, model *finance.SampathModel) error
}

type Config struct {
	Logger  zerolog.Logger
	Storage storage.MessageStorage[*finance.SampathModel]
}

type sampathBillService struct {
	logger  zerolog.Logger
	storage storage.MessageStorage[*finance.SampathModel]
}

func NewSampathBillService(c *Config) SampathBillService {
	return &sampathBillService{
		logger:  c.Logger,
		storage: c.Storage,
	}
}

func (s *sampathBillService) HandleSampathBill(ctx context.Context, model *finance.SampathModel) error {
	s.logger.Info().Msgf("Handling Sampath bill: %s", model.TransactionType)

	if err := s.storage.Save(ctx, model); err != nil {
		s.logger.Error().Err(err).Msg("Failed to save Sampath bill")
		return err
	}

	s.logger.Info().Msg("Sampath bill saved successfully")

	return nil
}
