package autofinance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
	"google.golang.org/api/sheets/v4"
)

const testSender = "TEST_SENDER"

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

	var req Request
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		app.logger.Error().Err(err).Msg("Failed to unmarshal request")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	app.logger.Debug().Ctx(ctx).Any("request", req).Msg("Request received")
	if req.Test || req.Sender == testSender {
		app.logger.Info().Msg("Test mode is enabled, skipping sheet write")
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "Test mode, no action taken",
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
