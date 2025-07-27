package ebill

import (
	"context"

	"auto-finance/internal/models/ebill"
	"auto-finance/internal/storage"

	"github.com/rs/zerolog"
)

type LECOBillService interface {
	HandleLECOBill(ctx context.Context, bill *ebill.ElectricityBill) error
}

type Config struct {
	Logger  zerolog.Logger
	Storage storage.MessageStorage[*ebill.ElectricityBill]
}

type lecoBillService struct {
	logger  zerolog.Logger
	storage storage.MessageStorage[*ebill.ElectricityBill]
}

func NewLECOBillService(c *Config) LECOBillService {
	return &lecoBillService{
		logger:  c.Logger,
		storage: c.Storage,
	}
}

func (s *lecoBillService) HandleLECOBill(ctx context.Context, bill *ebill.ElectricityBill) error {
	s.logger.Info().Msgf("Handling LECO bill: %s", bill.AccountName)

	if err := s.storage.Save(ctx, bill); err != nil {
		s.logger.Error().Err(err).Msg("Failed to save LECO bill")
		return err
	}

	s.logger.Info().Msg("LECO bill saved successfully")

	return nil
}
