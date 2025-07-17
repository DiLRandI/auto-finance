package autofinance

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
)

type Config struct {
	Logger *slog.Logger
}
type App struct {
	logger *slog.Logger
}

func New(config *Config) *App {
	return &App{
		logger: config.Logger,
	}
}

func (app *App) Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	app.logger.DebugContext(ctx, "Received request", slog.String("event", event.Body))
	defer app.logger.DebugContext(ctx, "Handler finished")

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Hello from Auto Finance!",
	}, nil
}
