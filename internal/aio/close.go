package aio

import (
	"io"

	"github.com/rs/zerolog/log"
)

// Close will close the given closer and log the error if required.
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Error().Err(err).Msgf("close failed")
	}
}
