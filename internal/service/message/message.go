package message

import (
	"context"
	"errors"
	"fmt"

	ebillModel "auto-finance/internal/models/ebill"
	"auto-finance/internal/service/ebill"
	"auto-finance/internal/smsparser"

	"github.com/rs/zerolog"
)

type Message struct {
	Sender string `json:"sender"`
	Body   string `json:"body"`
}

type Service interface {
	PassMessage(ctx context.Context, msg Message) error
}
type Config struct {
	Logger          zerolog.Logger
	Parsers         []smsparser.UniversalParser
	LecoBillService ebill.LECOBillService
}
type service struct {
	logger          zerolog.Logger
	parsers         []smsparser.UniversalParser
	lecoBillService ebill.LECOBillService
}

func New(c *Config) Service {
	return &service{
		logger:          c.Logger,
		parsers:         c.Parsers,
		lecoBillService: c.LecoBillService,
	}
}

func (s *service) PassMessage(ctx context.Context, msg Message) error {
	s.logger.Info().Ctx(ctx).Str("sender", msg.Sender).Msg("Processing message")

	parseErrors := make([]error, 0)
	for _, parser := range s.parsers {
		obj, err := parser.Parse(msg.Body)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}

		if obj == nil {
			parseErrors = append(parseErrors, fmt.Errorf("parser %s returned nil for message: %s", parser.GetName(), msg.Body))
		}

		parseErrors = nil // reset errors if we successfully parsed at least one message

		switch v := obj.(type) {
		case *ebillModel.ElectricityBill:
			if err := s.lecoBillService.HandleLECOBill(ctx, v); err != nil {
				s.logger.Error().Err(err).Msg("Failed to handle LECO bill")
				return fmt.Errorf("failed to handle LECO bill: %w", err)
			}
		default:
			s.logger.Warn().Msgf("Unknown object type: %T", v)
		}

	}

	return errors.Join(parseErrors...)
}
