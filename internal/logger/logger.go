package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetupConsoleLogger() {
	log.Logger = zerolog.New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger()
}

func SetLogLevel(logLevel string) error {
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	return nil
}
