package main

import (
	"context"
	"flag"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/timer"
	"github.com/rs/zerolog/log"
	"runtime"
	"strings"
	"time"
)

var downloadUrl string

func init() {
	logger.SetupConsoleLogger()

	var configPath string
	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the config file")
	flag.StringVar(&downloadUrl, "url", "", "download url of the json or zip file")

	flag.Parse()

	err := config.Load(configPath)
	if err != nil {
		panic(err)
	}
	cfg := config.Get()

	if downloadUrl == "" {
		downloadUrl = config.Get().Mtgjson.DownloadURL
	}

	err = logger.SetLogLevel(cfg.Logging.LevelOrDefault())
	if err != nil {
		panic(err)
	}

	log.Info().Msgf("OS\t\t %s", runtime.GOOS)
	log.Info().Msgf("ARCH\t\t %s", runtime.GOARCH)
	log.Info().Msgf("CPUs\t\t %d", runtime.NumCPU())
	log.Info().Msgf("Starting with url %s", downloadUrl)
}

func main() {
	defer timer.TimeTrack(time.Now(), "import")

	importer, closeConn, err := buildImporter()
	if err != nil {
		log.Error().Stack().Err(err).Msg("failed to build importer instance")
		return
	}
	defer closeConn()

	r, err := importer.Import(strings.NewReader(downloadUrl))
	if err != nil {
		log.Error().Stack().Err(err).Msg("import failed")
		return
	}

	log.Info().Msgf("Report %#v", r)
	logMemUsage()
}

func buildImporter() (api.Importer, func() error, error) {
	conn, err := postgres.Connect(context.Background(), &config.Get().Database)
	if err != nil {
		return nil, nil, err
	}

	csDao := cardset.NewDao(conn)
	cDao := card.NewDao(conn)
	csService := cardset.NewService(csDao)
	cService := card.NewService(cDao)

	imp := mtgjson.NewImporter(csService, cService)

	return mtgjson.NewDownloadableImport(imp), conn.Close, nil
}

// printMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func logMemUsage() uint64 {
	bToMB := func(b uint64) uint64 {
		return b / 1024 / 1024
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Info().Msgf("Alloc = %v MiB\tHeapAlloc  = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v", bToMB(m.Alloc), bToMB(m.HeapAlloc), bToMB(m.TotalAlloc), bToMB(m.Sys), m.NumGC)

	return m.Alloc
}
