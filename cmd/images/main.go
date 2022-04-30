package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/images"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/konstantinfoerster/card-importer-go/internal/stats"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/timer"
	"github.com/rs/zerolog/log"
	"runtime"
	"time"
)

const usage = `Usage: card-images-cli [options...]
  -c, --config path to the config file (default: ./configs/application.yaml)
  -p, --page start page number (default: 1)
  -s, --size amount of entries per page (default: 20)
  -h, --help prints help information
`

var configPath string
var pageConfig images.PageConfig

func init() {
	logger.SetupConsoleLogger()

	flag.StringVar(&configPath, "c", "./configs/application.yaml", "path to the config file")
	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the config file")
	flag.IntVar(&pageConfig.Page, "p", 1, "start page number")
	flag.IntVar(&pageConfig.Page, "page", 1, "start page number")
	flag.IntVar(&pageConfig.Size, "s", 20, "amount of entries per page")
	flag.IntVar(&pageConfig.Size, "size", 20, "amount of entries per page")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	err := config.Load(configPath)
	if err != nil {
		panic(err)
	}
	cfg := config.Get()

	err = logger.SetLogLevel(cfg.Logging.LevelOrDefault())
	if err != nil {
		panic(err)
	}

	log.Info().Msgf("OS\t\t %s", runtime.GOOS)
	log.Info().Msgf("ARCH\t\t %s", runtime.GOARCH)
	log.Info().Msgf("CPUs\t\t %d", runtime.NumCPU())
}

func main() {
	defer timer.TimeTrack(time.Now(), "images")

	cfg := config.Get()

	store, err := storage.NewLocalStorage(cfg.Storage)
	if err != nil {
		log.Error().Err(err).Msg("failed to create local storage")
		return
	}
	fetcher := fetch.NewDefaultFetcher()

	conn, err := postgres.Connect(context.Background(), cfg.Database)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to the database")
		return
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Error().Err(cErr).Msgf("Failed to close database connection")
		}
	}(conn.Close)

	cardDao := card.NewDao(conn)

	report, err := images.NewImporter(cardDao, store, scryfall.NewProcessor(cfg.Scryfall, fetcher)).Import(pageConfig)
	if err != nil {
		log.Error().Err(err).Msg("image import failed")
		return
	}
	log.Info().Msgf("Report %#v", report)

	stats.LogMemUsage()
}
