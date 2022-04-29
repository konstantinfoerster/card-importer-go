package stats

import (
	"github.com/rs/zerolog/log"
	"runtime"
)

// LogMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func LogMemUsage() uint64 {
	bToMB := func(b uint64) uint64 {
		return b / 1024 / 1024
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Info().Msgf("Alloc = %v MiB\tHeapAlloc  = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v", bToMB(m.Alloc), bToMB(m.HeapAlloc), bToMB(m.TotalAlloc), bToMB(m.Sys), m.NumGC)

	return m.Alloc
}
