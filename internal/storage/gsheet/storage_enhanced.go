package gsheet

import (
	"context"
	"fmt"
	"time"

	"auto-finance/internal/errors"
	"auto-finance/internal/models"
	"auto-finance/internal/utils/retry"

	"github.com/google/uuid"
	"google.golang.org/api/sheets/v4"
)

// EnhancedGSheetStorage provides Google Sheets storage with retry capabilities
type EnhancedGSheetStorage struct {
	service           *sheets.Service
	sheetID           string
	sheetName         string
	googleRetryConfig retry.GoogleRetryConfig
}

// EnhancedConfig contains configuration for enhanced Google Sheets storage
type EnhancedConfig struct {
	Service           *sheets.Service
	SheetID           string
	SheetName         string
	GoogleRetryConfig retry.GoogleRetryConfig
}

// NewEnhanced creates a new enhanced Google Sheets storage with retry capabilities
func NewEnhanced(config *EnhancedConfig) *EnhancedGSheetStorage {
	retryConfig := config.GoogleRetryConfig
	if retryConfig.MaxAttempts == 0 {
		retryConfig = retry.DefaultGoogleRetryConfig()
	}

	return &EnhancedGSheetStorage{
		service:           config.Service,
		sheetID:           config.SheetID,
		sheetName:         config.SheetName,
		googleRetryConfig: retryConfig,
	}
}

// Save saves a message to Google Sheets with retry logic
func (s *EnhancedGSheetStorage) Save(ctx context.Context, message *models.Message) error {
	operation := func() error {
		var vr sheets.ValueRange
		vr.Values = append(vr.Values, []interface{}{
			message.ID.String(),
			message.From,
			message.Message,
			message.Time.Format("2006-01-02 15:04:05"),
		})

		_, err := s.service.Spreadsheets.Values.Append(
			s.sheetID,
			s.sheetName,
			&vr,
		).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
		if err != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to append message to sheet: %w", err),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}
		return nil
	}

	return retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
}

// Read reads a message from Google Sheets with retry logic
func (s *EnhancedGSheetStorage) Read(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var message *models.Message
	var err error

	operation := func() error {
		rowIndex, findErr := s.findRowByID(ctx, id)
		if findErr != nil {
			return findErr
		}
		if rowIndex == -1 {
			return nil // Not found, not an error
		}

		readRange := fmt.Sprintf("%s!A%d:D%d", s.sheetName, rowIndex, rowIndex)
		resp, getErr := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
		if getErr != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to read message from sheet: %w", getErr),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}

		if len(resp.Values) == 0 || len(resp.Values[0]) < 4 {
			return fmt.Errorf("message not found or row is malformed")
		}

		row := resp.Values[0]
		msgID, parseErr := uuid.Parse(row[0].(string))
		if parseErr != nil {
			return fmt.Errorf("failed to parse message ID: %w", parseErr)
		}

		t, timeErr := time.Parse("2006-01-02 15:04:05", row[3].(string))
		if timeErr != nil {
			return fmt.Errorf("failed to parse time: %w", timeErr)
		}

		message = &models.Message{
			ID:      msgID,
			From:    row[1].(string),
			Message: row[2].(string),
			Time:    t,
		}
		return nil
	}

	err = retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
	if err != nil {
		return nil, err
	}

	return message, nil
}

// ReadAll reads all messages from Google Sheets with retry logic
func (s *EnhancedGSheetStorage) ReadAll(ctx context.Context, pageSize, pageNumber int) ([]*models.Message, error) {
	var messages []*models.Message

	operation := func() error {
		readRange := fmt.Sprintf("%s!A%d:D%d", s.sheetName, pageNumber*pageSize+1, (pageNumber+1)*pageSize)
		resp, err := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
		if err != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to read messages from sheet: %w", err),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}

		messages = make([]*models.Message, 0, len(resp.Values))
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
		return nil
	}

	err := retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// Delete deletes a message from Google Sheets with retry logic
func (s *EnhancedGSheetStorage) Delete(ctx context.Context, id uuid.UUID) error {
	operation := func() error {
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
			return errors.NewRetryableError(
				fmt.Errorf("failed to delete row from sheet: %w", err),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}

		return nil
	}

	return retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
}

// findRowByID finds a row by message ID with retry logic
func (s *EnhancedGSheetStorage) findRowByID(ctx context.Context, id uuid.UUID) (int, error) {
	rowIndex := -1

	operation := func() error {
		readRange := fmt.Sprintf("%s!A:A", s.sheetName)
		resp, err := s.service.Spreadsheets.Values.Get(s.sheetID, readRange).Do()
		if err != nil {
			return errors.NewRetryableError(
				fmt.Errorf("failed to read IDs from sheet: %w", err),
				errors.ErrorTypeGoogle,
				2*time.Second,
				3,
			)
		}

		for i, row := range resp.Values {
			if len(row) > 0 && row[0].(string) == id.String() {
				rowIndex = i + 1 // 1-based index
				return nil
			}
		}

		rowIndex = -1 // Not found
		return nil
	}

	err := retry.WithGoogleRetry(ctx, s.googleRetryConfig, operation)
	if err != nil {
		return -1, err
	}

	return rowIndex, nil
}
