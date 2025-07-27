package autofinance

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
	"google.golang.org/api/sheets/v4"
)

type Config struct {
	Logger zerolog.Logger
	SS     *sheets.Service
}
type App struct {
	logger zerolog.Logger
	srv    *sheets.Service
}

func New(config *Config) *App {
	return &App{
		logger: config.Logger,
		srv:    config.SS,
	}
}

func (app *App) Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	app.logger.Debug().Ctx(ctx).Any("event", event).Msg("Handler started")
	defer app.logger.Debug().Ctx(ctx).Msg("Handler finished")

	dataToWrite := [][]interface{}{
		{"Name", "Email", "Date Joined"},
		{"Dee", "alice@example.com", "2023-01-15"},
		{"Bob Johnson", "bob@example.com", "2023-02-20"},
		{"Charlie Brown", "charlie@example.com", "2023-03-10"},
	}

	spreadsheetID := "1sM6wKz2pVlus-fZ8qbDCGmJBYqRxwE3XfhLv1kn1-J4"

	// Replace with the name of the sheet and the range where you want to write data.
	// For example, "Sheet1!A1" will start appending from cell A1 of Sheet1.
	sheetRange := "Sheet1!A1"

	// Write the data to the Google Sheet.
	err := app.writeToSheet(spreadsheetID, sheetRange, dataToWrite)
	if err != nil {
		app.logger.Error().Ctx(ctx).Msg("Error writing data to sheet")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Hello from Auto Finance!",
	}, nil
}

func (app *App) writeToSheet(spreadsheetID, sheetRange string, data [][]interface{}) error {
	valueRange := &sheets.ValueRange{
		Values: data,
	}

	// Call the Append method to add data to the sheet.
	// The "USER_ENTERED" value input option means that the data will be parsed
	// as if it were entered by a user (e.g., numbers will be parsed as numbers, dates as dates).
	// The "INSERT_ROWS" insert data option means new rows will be inserted at the end of the sheet.
	_, err := app.srv.Spreadsheets.Values.Append(spreadsheetID, sheetRange, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	return nil
}
