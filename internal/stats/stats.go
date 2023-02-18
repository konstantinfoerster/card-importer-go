package stats

import (
	"runtime"

	"github.com/rs/zerolog/log"
)

// LogMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func LogMemUsage() uint64 {
	var oneKiB uint64 = 1024
	bToMiB := func(b uint64) uint64 {
		return b / oneKiB / oneKiB
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Info().Msgf("Alloc = %v MiB\tHeapAlloc  = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v",
		bToMiB(m.Alloc), bToMiB(m.HeapAlloc), bToMiB(m.TotalAlloc), bToMiB(m.Sys), m.NumGC)

	return m.Alloc
}
