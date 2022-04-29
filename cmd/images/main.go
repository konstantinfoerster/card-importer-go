package main

import (
	"context"
	"flag"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
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

func init() {
	logger.SetupConsoleLogger()

	var configPath string

	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the config file")

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

	report, err := scryfall.NewImporter(cfg.Scryfall, fetcher, store, cardDao).Import()
	if err != nil {
		log.Error().Err(err).Msg("image import failed")
		return
	}
	log.Info().Msgf("Report %#v", report)

	stats.LogMemUsage()
}
