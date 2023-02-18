package timer

import (
	"time"

	"github.com/rs/zerolog/log"
)

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Info().Msgf("%s took %s", name, elapsed)
}
