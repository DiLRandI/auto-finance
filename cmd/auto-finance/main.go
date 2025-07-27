package main

import (
	"context"
	"fmt"
	"os"

	autofinance "auto-finance/internal/app/auto-finance"
	appConfig "auto-finance/internal/config"
	"auto-finance/internal/logger"
	parameterstore "auto-finance/internal/parameter-store"
	"auto-finance/internal/service/ebill"
	"auto-finance/internal/service/message"
	"auto-finance/internal/smsparser"
	"auto-finance/internal/smsparser/bill/leco"
	ebillStorage "auto-finance/internal/storage/ebill"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var version = "local"

func main() {
	ctx := context.Background()

	logger, err := logger.NewLogger(version, os.Getenv("LOG_LEVEL"))
	if err != nil {
		panic(err)
	}

	logger.Info().Msg("Initializing Auto Finance Lambda function")
	defer logger.Info().Msg("Auto Finance Lambda function initialization complete")

	appConfig, err := appConfig.LoadConfig()
	if err != nil {
		logger.Err(err).Msg("Failed to load application config")
		os.Exit(1)
	}

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

	msgSvc := message.New(&message.Config{
		Logger: logger,
		Parsers: []smsparser.UniversalParser{
			smsparser.NewGenericParserWrapper(leco.New()),
		},
		LecoBillService: ebill.NewLECOBillService(&ebill.Config{
			Logger: logger,
			Storage: ebillStorage.New(&ebillStorage.Config{
				Service:   srv,
				SheetID:   appConfig.LecoSheetConfig.SheetID,
				SheetName: appConfig.LecoSheetConfig.SheetName,
			}),
		}),
	})

	app := autofinance.New(&autofinance.Config{
		Logger:         logger,
		MessageService: msgSvc,
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
