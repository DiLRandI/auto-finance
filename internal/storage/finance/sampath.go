package finance

import (
	"context"
	"fmt"
	"time"

	"auto-finance/internal/errors"
	"auto-finance/internal/models/finance"
	"auto-finance/internal/utils/retry"

	"auto-finance/internal/storage"

	"google.golang.org/api/sheets/v4"
)

// SmpathStorage provides LECO bill storage with retry capabilities
type SmpathStorage struct {
	service           *sheets.Service
	sheetID           string
	sheetName         string
	googleRetryConfig retry.GoogleRetryConfig
}

// SampathConfig contains configuration for enhanced LECO bill storage
type SampathConfig struct {
	Service           *sheets.Service
	SheetID           string
	SheetName         string
	GoogleRetryConfig *retry.GoogleRetryConfig
}

// NewSampathStorage creates a new enhanced LECO bill storage with retry capabilities
func NewSampathStorage(config *SampathConfig) storage.MessageStorage[*finance.SampathModel] {
	retryConfig := retry.DefaultGoogleRetryConfig()
	if config.GoogleRetryConfig != nil {
		retryConfig = *config.GoogleRetryConfig
	}

	return &SmpathStorage{
		service:           config.Service,
		sheetID:           config.SheetID,
		sheetName:         config.SheetName,
		googleRetryConfig: retryConfig,
	}
}

// Save saves an electricity bill to Google Sheets with retry logic
func (s *SmpathStorage) Save(ctx context.Context, bill *finance.SampathModel) error {
	operation := func() error {
		var vr sheets.ValueRange
		vr.Values = append(vr.Values, []interface{}{
			bill.SmsDateTime,
			bill.Amount,
			bill.Currency,
			bill.Status,
			bill.TransactionType,
			bill.Identifier,
			bill.Merchant,
			bill.AvailableBalance,
			bill.AvailableBalanceCurrency,
		})

		_, err := s.service.Spreadsheets.Values.Append(
			s.sheetID,
			s.sheetName,
			&vr,
		).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
		if err != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to append sampath statement to sheet: %w", err),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}
		return nil
	}

	return retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
}
