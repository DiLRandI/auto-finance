package config

import (
	"auto-finance/internal/storage"
	"context"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LecoSheetConfig    SheetConfig `toml:"leco_sheet_config"`
	FinanceSheetConfig SheetConfig `toml:"finance_sheet_config"`
}

type SheetConfig struct {
	SheetID   string `toml:"sheet_id"`
	SheetName string `toml:"sheet_name"`
}

func LoadConfig(storage storage.ConfigStorage) (*Config, error) {
	var config Config

	data, err := storage.GetConfig(context.Background(), "config.toml")
	if err != nil {
		return nil, err
	}

	_, err = toml.Decode(string(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
