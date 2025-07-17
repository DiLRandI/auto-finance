package main

import (
	"log/slog"
	"os"

	autofinance "auto-finance/inernal/app/auto-finance"

	"github.com/aws/aws-lambda-go/lambda"
)

var version = "local"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(os.Getenv("LOG_LEVEL")),
	})).With("version", version)

	logger.Info("Starting Auto Finance Lambda function")
	defer logger.Info("Auto Finance Lambda function finished")

	app := autofinance.New(&autofinance.Config{
		Logger: logger,
	})

	lambda.Start(app.Handler)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
