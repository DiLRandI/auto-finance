package main

import (
	autoebill "auto-finance/inernal/app/auto-ebill"
	"auto-finance/inernal/logger"
	parameterstore "auto-finance/inernal/parameter-store"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var version = "local"

func main() {
	ctx := context.Background()

	logger, err := logger.NewLogger("AutoEBill", version, os.Getenv("LOG_LEVEL"))
	if err != nil {
		panic(err)
	}

	logger.Info().Msg("Initializing Auto EBill Lambda function")
	defer logger.Info().Msg("Auto EBill Lambda function initialization complete")

	sheetKey, err := loadParameters(ctx)
	if err != nil {
		logger.Err(err).Msg("Failed to load parameters from Parameter Store")
		os.Exit(1)
	}

	srv, err := sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsScope), option.WithCredentialsJSON([]byte(sheetKey)))
	if err != nil {
		logger.Err(err).Msg("Failed to create Sheets service")
		os.Exit(1)
	}

	app := autoebill.New(&autoebill.Config{
		Logger: logger,
		SS:     srv,
	})

	lambda.Start(app.Handler)
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
