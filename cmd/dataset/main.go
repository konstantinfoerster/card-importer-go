package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/logger"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
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

const usage = `Usage: card-dataset-cli [options...]
  --config path to the configuration file
  --file path to local dataset json file, has precedence over the url flag or configuration file
  --help prints help information
`

func setup() (*url.URL, config.Config) {
	logger.SetupConsoleLogger()

	var configPaths arrayFlag
	var file string

	flag.Var(&configPaths, "config", "path to the configuration files e.g. --config /config.yaml --config /secret.yaml")
	flag.StringVar(&file, "file", "",
		"path to local dataset json file, has precedence over the configuration file")
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

	if file == "" {
		downloadURL := cfg.Mtgjson.DatasetURL
		log.Info().Msgf("Using dataset from url %s", downloadURL)
		u, pErr := url.Parse(downloadURL)
		if pErr != nil {
			panic(pErr)
		}

		return u, cfg
	}

	log.Info().Msgf("Using dataset from file %s", file)
	u, pErr := url.Parse(file)
	if pErr != nil {
		panic(pErr)
	}

	return u, cfg
}

func main() {
	defer timer.TimeTrack(time.Now(), "import")

	datasetSource, cfg := setup()

	conn, err := postgres.Connect(context.Background(), cfg.Database)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to the database")

		return
	}
	defer func(toCloseFn func() error) {
		cErr := toCloseFn()
		if cErr != nil {
			log.Panic().Err(cErr).Msg("Failed to close database connection")
		}
	}(conn.Close)

	csService := cards.NewSetService(cards.NewSetDao(conn))
	cService := cards.NewCardService(cards.NewCardDao(conn))
	imp := mtgjson.NewImporter(csService, cService)

	store, err := storage.NewLocalStorage(cfg.Storage)
	if err != nil {
		log.Panic().Err(err).Msg("failed to create local storage")

		return
	}

	c := &http.Client{
		Timeout: cfg.Mtgjson.Client.Timeout,
	}
	client := web.NewClient(cfg.Mtgjson.Client, c)
	loader := mtgjson.NewLoader(imp, cfg.Mtgjson, client, store)
	report, iErr := loader.Load(datasetSource)
	if iErr != nil {
		log.Panic().Err(iErr).Msg("dataset import failed")

		return
	}

	log.Info().Msgf("Report %#v", report)
}
