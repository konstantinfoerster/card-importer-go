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
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/timer"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/rs/zerolog/log"
)

const usage = `Usage: card-images-cli [options...]
  -c, --config path to the configuration file (default: ./configs/application.yaml)
  -p, --page start page number (default: 1)
  -s, --size amount of entries per page (default: 20)
  -h, --help prints help information
`

func setup() (cards.PageConfig, *config.Config) {
	logger.SetupConsoleLogger()

	var configPath string
	var pageConfig cards.PageConfig

	flag.StringVar(&configPath, "c", "./configs/application.yaml", "path to the configuration file")
	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the configuration file")
	flag.IntVar(&pageConfig.Page, "p", 1, "start page number")
	flag.IntVar(&pageConfig.Page, "page", 1, "start page number")
	flag.IntVar(&pageConfig.Size, "s", 20, "amount of entries per page")
	flag.IntVar(&pageConfig.Size, "size", 20, "amount of entries per page")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	cfg, err := config.Load(configPath)
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
		log.Error().Err(err).Msg("failed to create local storage")

		return
	}

	conn, err := postgres.Connect(context.Background(), cfg.Database)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to the database")

		return
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Error().Err(cErr).Msgf("Failed to close database connection")

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

		log.Info().Msgf("image import exit ...")

		done <- true
	}()

	go func() {
		defer func() {
			done <- true
		}()

		report, err := importer.Import(pageConfig)
		if err != nil {
			log.Error().Err(err).Msg("image import failed")

			return
		}
		log.Info().Msgf("Report %#v", report)
	}()

	<-done
}
