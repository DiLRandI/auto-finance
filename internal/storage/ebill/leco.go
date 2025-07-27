package ebill

import (
	"context"
	"fmt"

	"auto-finance/internal/models/ebill"
	"auto-finance/internal/storage"

	"google.golang.org/api/sheets/v4"
)

type lecoStorage struct {
	service   *sheets.Service
	sheetID   string
	sheetName string
}

type Config struct {
	Service   *sheets.Service
	SheetID   string
	SheetName string
}

func New(config *Config) storage.MessageStorage[*ebill.ElectricityBill] {
	return &lecoStorage{
		service:   config.Service,
		sheetID:   config.SheetID,
		sheetName: config.SheetName,
	}
}

func (s *lecoStorage) Save(ctx context.Context, message *ebill.ElectricityBill) error {
	var vr sheets.ValueRange

	vr.Values = append(vr.Values, []interface{}{
		message.AccountNumber,
		message.AccountType,
		message.AccountName,
		message.ReadOn,
		message.ImportPrevious,
		message.ImportCurrent,
		message.ImportUnits,
		message.ExportPrevious,
		message.ExportCurrent,
		message.ExportUnits,
		message.NetUnits,
		message.NetUnitsType,
		message.MonthlyBill,
		message.OtherCharges,
		message.SSCL,
		message.OpeningBalance,
		message.OpeningBalanceDate,
		message.TotalPayable,
		message.LastPaymentAmount,
		message.LastPaymentDate,
		message.LastGenPayment,
	})

	_, err := s.service.Spreadsheets.Values.Append(s.sheetID, s.sheetName, &vr).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to append message to sheet: %w", err)
	}
	return nil
}
