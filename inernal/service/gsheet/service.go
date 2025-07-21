package gsheet

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/sheets/v4"
)

type SheetWriter interface {
	WriteRecord(ctx context.Context, date time.Time, message string) error
}

type gsheetService struct {
	service   *sheets.Service
	sheetID   string
	sheetName string
}

func NewGSheetService(svc *sheets.Service, sheetID, sheetName string) SheetWriter {
	return &gsheetService{
		service:   svc,
		sheetID:   sheetID,
		sheetName: sheetName,
	}
}

func (s *gsheetService) WriteRecord(ctx context.Context, date time.Time, message string) error {
	values := [][]interface{}{
		{date.Format(time.RFC3339), message},
	}

	_, err := s.service.Spreadsheets.Values.Append(
		s.sheetID,
		fmt.Sprintf("%s!A:B", s.sheetName),
		&sheets.ValueRange{
			Values: values,
		},
	).ValueInputOption("RAW").Context(ctx).Do()

	return err
}
