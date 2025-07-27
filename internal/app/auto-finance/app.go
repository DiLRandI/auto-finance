package autofinance

import (
	"context"
	"encoding/json"

	"auto-finance/internal/service/message"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
)

const testSender = "TEST_SENDER"

type Config struct {
	Logger         zerolog.Logger
	MessageService message.Service
}
type App struct {
	logger         zerolog.Logger
	messageService message.Service
}

func New(config *Config) *App {
	return &App{
		logger:         config.Logger,
		messageService: config.MessageService,
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

	app.logger.Info().Ctx(ctx).Any("request", req).Msg("Request received")

	if req.Test || req.Sender == testSender {
		app.logger.Info().Msg("Test mode is enabled, skipping sheet write")
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "Test mode, no action taken",
		}, nil
	}

	if err := app.messageService.PassMessage(ctx, message.Message{
		Sender: req.Sender,
		Body:   req.Body,
	}); err != nil {
		app.logger.Error().Err(err).Msg("Failed to pass message")
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
