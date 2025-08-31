package ebill

import (
	"auto-finance/internal/errors"
	"auto-finance/internal/models/ebill"
	"auto-finance/internal/utils/retry"
	"context"
	"fmt"
	"time"

	"auto-finance/internal/storage"

	"google.golang.org/api/sheets/v4"
)

// EnhancedLECOStorage provides LECO bill storage with retry capabilities
type EnhancedLECOStorage struct {
	service           *sheets.Service
	sheetID           string
	sheetName         string
	googleRetryConfig retry.GoogleRetryConfig
}

// EnhancedConfig contains configuration for enhanced LECO bill storage
type EnhancedConfig struct {
	Service           *sheets.Service
	SheetID           string
	SheetName         string
	GoogleRetryConfig *retry.GoogleRetryConfig
}

// NewEnhanced creates a new enhanced LECO bill storage with retry capabilities
func NewEnhanced(config *EnhancedConfig) storage.MessageStorage[*ebill.ElectricityBill] {
	retryConfig := retry.DefaultGoogleRetryConfig()
	if config.GoogleRetryConfig != nil {
		retryConfig = *config.GoogleRetryConfig
	}

	return &EnhancedLECOStorage{
		service:           config.Service,
		sheetID:           config.SheetID,
		sheetName:         config.SheetName,
		googleRetryConfig: retryConfig,
	}
}

// Save saves an electricity bill to Google Sheets with retry logic
func (s *EnhancedLECOStorage) Save(ctx context.Context, bill *ebill.ElectricityBill) error {
	operation := func() error {
		var vr sheets.ValueRange
		vr.Values = append(vr.Values, []interface{}{
			bill.AccountNumber,
			bill.AccountType,
			bill.AccountName,
			bill.ReadOn,
			bill.ImportPrevious,
			bill.ImportCurrent,
			bill.ImportUnits,
			bill.ExportPrevious,
			bill.ExportCurrent,
			bill.ExportUnits,
			bill.NetUnits,
			bill.NetUnitsType,
			bill.MonthlyBill,
			bill.OtherCharges,
			bill.SSCL,
			bill.OpeningBalance,
			bill.OpeningBalanceDate,
			bill.TotalPayable,
			bill.LastPaymentAmount,
			bill.LastPaymentDate,
			bill.LastGenPayment,
		})

		_, err := s.service.Spreadsheets.Values.Append(
			s.sheetID,
			s.sheetName,
			&vr,
		).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()

		if err != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to append electricity bill to sheet: %w", err),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}
		return nil
	}

	return retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
}
