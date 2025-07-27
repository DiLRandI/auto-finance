package gsheet

import (
	"context"
	"fmt"
	"time"

	"auto-finance/internal/models"

	"github.com/google/uuid"
	"google.golang.org/api/sheets/v4"
)

type gsheetStorage struct {
	service   *sheets.Service
	sheetID   string
	sheetName string
}

type Config struct {
	Service   *sheets.Service
	SheetID   string
	SheetName string
}

func New(config *Config) *gsheetStorage {
	return &gsheetStorage{
		service:   config.Service,
		sheetID:   config.SheetID,
		sheetName: config.SheetName,
	}
}

func (s *gsheetStorage) Save(ctx context.Context, message *models.Message) error {
	var vr sheets.ValueRange
	vr.Values = append(vr.Values, []interface{}{message.ID.String(), message.From, message.Message, message.Time.Format("2006-01-02 15:04:05")})
	_, err := s.service.Spreadsheets.Values.Append(s.sheetID, s.sheetName, &vr).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to append message to sheet: %w", err)
	}
	return nil
}

func (s *gsheetStorage) Read(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	rowIndex, err := s.findRowByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rowIndex == -1 {
		return nil, nil // Not found
	}

	readRange := fmt.Sprintf("%s!A%d:D%d", s.sheetName, rowIndex, rowIndex)
	resp, err := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read message from sheet: %w", err)
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) < 4 {
		return nil, fmt.Errorf("message not found or row is malformed")
	}

	row := resp.Values[0]
	msgID, err := uuid.Parse(row[0].(string))
	if err != nil {
		return nil, fmt.Errorf("failed to parse message ID: %w", err)
	}
	t, err := time.Parse("2006-01-02 15:04:05", row[3].(string))
	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %w", err)
	}

	return &models.Message{
		ID:      msgID,
		From:    row[1].(string),
		Message: row[2].(string),
		Time:    t,
	}, nil
}

func (s *gsheetStorage) ReadAll(ctx context.Context, pageSize, pageNumber int) ([]*models.Message, error) {
	readRange := fmt.Sprintf("%s!A%d:D%d", s.sheetName, pageNumber*pageSize+1, (pageNumber+1)*pageSize)
	resp, err := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read messages from sheet: %w", err)
	}

	var messages []*models.Message
	for _, row := range resp.Values {
		if len(row) < 4 {
			continue
		}
		id, err := uuid.Parse(row[0].(string))
		if err != nil {
			continue
		}
		t, err := time.Parse("2006-01-02 15:04:05", row[3].(string))
		if err != nil {
			continue
		}
		messages = append(messages, &models.Message{
			ID:      id,
			From:    row[1].(string),
			Message: row[2].(string),
			Time:    t,
		})
	}
	return messages, nil
}

func (s *gsheetStorage) Delete(ctx context.Context, id uuid.UUID) error {
	rowIndex, err := s.findRowByID(ctx, id)
	if err != nil {
		return err
	}
	if rowIndex == -1 {
		return nil // Not found, nothing to delete
	}

	req := &sheets.Request{
		DeleteDimension: &sheets.DeleteDimensionRequest{
			Range: &sheets.DimensionRange{
				SheetId:    0, // Assuming the first sheet
				Dimension:  "ROWS",
				StartIndex: int64(rowIndex - 1),
				EndIndex:   int64(rowIndex),
			},
		},
	}

	batchUpdateReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{req},
	}

	_, err = s.service.Spreadsheets.BatchUpdate(s.sheetID, batchUpdateReq).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete row from sheet: %w", err)
	}

	return nil
}

func (s *gsheetStorage) findRowByID(ctx context.Context, id uuid.UUID) (int, error) {
	// This is inefficient and not recommended for large sheets.
	// A better approach would be to use a more suitable database.
	readRange := fmt.Sprintf("%s!A:A", s.sheetName)
	resp, err := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
	if err != nil {
		return -1, fmt.Errorf("failed to read IDs from sheet: %w", err)
	}

	for i, row := range resp.Values {
		if len(row) > 0 && row[0].(string) == id.String() {
			return i + 1, nil // 1-based index
		}
	}

	return -1, nil // Not found
}
