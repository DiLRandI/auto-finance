package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	autofinance "auto-finance/inernal/app/auto-finance"
	parameterstore "auto-finance/inernal/parameter-store"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var version = "local"

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(os.Getenv("LOG_LEVEL")),
	})).With("version", version)

	logger.Info("Starting Auto Finance Lambda function")
	defer logger.Info("Auto Finance Lambda function finished")

	sheetKey, err := loadParameters(ctx)
	if err != nil {
		logger.Error("Failed to load parameters", "error", err)
		os.Exit(1)
	}

	srv, err := sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsScope), option.WithCredentialsJSON([]byte(sheetKey)))
	if err != nil {
		logger.Error("Failed to create Sheets service", "error", err)
		os.Exit(1)
	}

	app := autofinance.New(&autofinance.Config{
		Logger: logger,
		SS:     srv,
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

func loadParameters(ctx context.Context) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := ssm.NewFromConfig(cfg)
	store := parameterstore.New(client)

	sk := os.Getenv("SHEET_KEY")

	return store.GetParameter(ctx, sk)
}
