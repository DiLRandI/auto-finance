package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func NewLogger(version, logLevel string) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		return zerolog.Logger{}, err
	}

	zerolog.SetGlobalLevel(level)

	return zerolog.New(os.Stdout).
		With().
		Str("version", version).
		Timestamp().
		Logger(), nil
}
