package config

import "github.com/BurntSushi/toml"

type Config struct {
	LecoSheetConfig SheetConfig `toml:"leco_sheet_config"`
}

type SheetConfig struct {
	SheetID   string `toml:"sheet_id"`
	SheetName string `toml:"sheet_name"`
}

func LoadConfig() (*Config, error) {
	var config Config
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
