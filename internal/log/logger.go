package log

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"os"
	"strings"
	"time"
)

func SetupConsoleLogger() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Stack().
		Caller().
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
