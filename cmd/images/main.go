package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/logger"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/timer"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/rs/zerolog/log"
)

type arrayFlag []string

func (a *arrayFlag) String() string {
	return fmt.Sprintf("%v", *a)
}

func (a *arrayFlag) Set(value string) error {
	*a = append(*a, value)

	return nil
}

const usage = `Usage: card-images-cli [options...]
  --config path to the configuration file
  --page start page number (default: 1)
  --size amount of entries per page (default: 20)
  --help prints help information
`

func setup() (cards.PageConfig, config.Config) {
	logger.SetupConsoleLogger()

	var configPaths arrayFlag
	var pageConfig cards.PageConfig

	flag.Var(&configPaths, "config", "path to the configuration files e.g. --config /config.yaml --config /secret.yaml")
	flag.IntVar(&pageConfig.Page, "page", 1, "start page number")
	flag.IntVar(&pageConfig.Size, "size", 20, "amount of entries per page")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	cfg, err := config.ReadConfigs(configPaths...)
	if err != nil {
		panic(err)
	}

	err = logger.SetLogLevel(cfg.Logging.LevelOrDefault())
	if err != nil {
		panic(err)
	}

	log.Info().Msgf("OS\t\t %s", runtime.GOOS)
	log.Info().Msgf("ARCH\t\t %s", runtime.GOARCH)
	log.Info().Msgf("CPUs\t\t %d", runtime.NumCPU())

	return pageConfig, cfg
}

func main() {
	defer timer.TimeTrack(time.Now(), "images import")

	pageConfig, cfg := setup()

	store, err := storage.NewLocalStorage(cfg.Storage)
	if err != nil {
		log.Panic().Err(err).Msg("failed to create local storage")

		return
	}

	conn, err := postgres.Connect(context.Background(), cfg.Database)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to the database")

		return
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Error().Err(cErr).Msg("Failed to close database connection")

			return
		}

		log.Info().Msg("closed database connection")
	}(conn.Close)

	cardDao := cards.NewCardDao(conn)

	client := &http.Client{
		Timeout: cfg.Scryfall.Client.Timeout,
	}
	wclient := web.NewClient(cfg.Scryfall.Client, client)
	sclient := scryfall.NewClient(cfg.Scryfall, wclient, scryfall.DefaultLanguages)
	importer := cards.NewImageImporter(cardDao, store, sclient)

	ctx := context.Background()
	done := make(chan bool, 1)
	nCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-nCtx.Done()

		log.Info().Msg("image import exit ...")

		done <- true
	}()

	go func() {
		defer func() {
			done <- true
		}()

		report, err := importer.Import(pageConfig)
		if err != nil {
			log.Panic().Err(err).Msg("image import failed")

			return
		}
		log.Info().Msgf("Report %#v", report)
	}()

	<-done
}
