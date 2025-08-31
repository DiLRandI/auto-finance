package main

import (
	"context"
	"fmt"
	"os"
	"time"

	autofinance "auto-finance/internal/app/auto-finance"
	appConfig "auto-finance/internal/config"
	"auto-finance/internal/logger"
	parameterstore "auto-finance/internal/parameter-store"
	"auto-finance/internal/service/ebill"
	"auto-finance/internal/service/message"
	"auto-finance/internal/smsparser"
	"auto-finance/internal/smsparser/banking/sampath"
	"auto-finance/internal/smsparser/bill/leco"
	ebillStorage "auto-finance/internal/storage/ebill"
	"auto-finance/internal/utils/retry"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
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

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to load AWS config: %w", err))
	}

	//TODO after observing pricing can remove s3

	// configStore := configStorage.New(&configStorage.Config{
	// 	Client: s3.NewFromConfig(awsConfig),
	// 	Bucket: os.Getenv("CONFIGURATION_BUCKET"),
	// 	RetryConfig: &retry.AWSRetryConfig{
	// 		MaxAttempts:    3,
	// 		InitialBackoff: 1 * time.Second,
	// 		MaxBackoff:     5 * time.Second,
	// 	},
	// })

	// appConfig, err := appConfig.LoadConfig(configStore)
	// if err != nil {
	// 	logger.Err(err).Msg("Failed to load application config")
	// 	os.Exit(1)
	// }

	params, err := loadParameters(ctx, awsConfig)
	if err != nil {
		logger.Err(err).Msg("Failed to load parameters from Parameter Store")
		os.Exit(1)
	}

	srv, err := sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsScope), option.WithCredentialsJSON([]byte(params.sheetKeys)))
	if err != nil {
		logger.Err(err).Msg("Failed to create Sheets service")
		os.Exit(1)
	}

	appConfig, err := appConfig.LoadConfigFromTomlBody([]byte(params.appConfig))
	if err != nil {
		logger.Err(err).Msg("Failed to load application config")
		os.Exit(1)
	}

	msgSvc := message.New(&message.Config{
		Logger: logger,
		Parsers: []smsparser.UniversalParser{
			smsparser.NewGenericParserWrapper(leco.New()),
			smsparser.NewGenericParserWrapper(sampath.New()),
		},
		LecoBillService: ebill.NewLECOBillService(&ebill.Config{
			Logger: logger,
			Storage: ebillStorage.NewEnhanced(&ebillStorage.EnhancedConfig{
				Service:   srv,
				SheetID:   appConfig.LecoSheetConfig.SheetID,
				SheetName: appConfig.LecoSheetConfig.SheetName,
				GoogleRetryConfig: &retry.GoogleRetryConfig{
					MaxAttempts:    3,
					InitialBackoff: 1 * time.Second,
					MaxBackoff:     5 * time.Second,
				},
			}),
		}),
	})

	app := autofinance.New(&autofinance.Config{
		Logger:         logger,
		MessageService: msgSvc,
	})

	lambda.Start(app.Handler)
}

type configParams struct {
	sheetKeys string
	appConfig string
}

func loadParameters(ctx context.Context, awsConfig aws.Config) (configParams, error) {
	client := ssm.NewFromConfig(awsConfig)

	// Use enhanced parameter store with retry capabilities
	store := parameterstore.NewWithConfig(&parameterstore.Config{
		Client: client,
		RetryConfig: &retry.AWSRetryConfig{
			MaxAttempts:    3,
			InitialBackoff: 1 * time.Second,
			MaxBackoff:     5 * time.Second,
		},
	})

	sk := os.Getenv("SHEET_KEY")
	ac := os.Getenv("APP_CONFIG")

	// Get parameters from the store
	sheetKey, err := store.GetParameter(ctx, sk)
	if err != nil {
		return configParams{}, fmt.Errorf("failed to get sheet key: %w", err)
	}

	appConfig, err := store.GetParameter(ctx, ac)
	if err != nil {
		return configParams{}, fmt.Errorf("failed to get app config: %w", err)
	}

	return configParams{
		sheetKeys: sheetKey,
		appConfig: appConfig,
	}, nil
}
