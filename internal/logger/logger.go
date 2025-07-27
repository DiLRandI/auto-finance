package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func NewLogger(appName, version, logLevel string) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		return zerolog.Logger{}, err
	}

	return zerolog.New(os.Stdout).
		With().
		Str("app", appName).
		Str("version", version).
		Timestamp().
		Logger().
		Level(level), nil
}
