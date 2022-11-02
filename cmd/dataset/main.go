package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/stats"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/timer"
	"github.com/rs/zerolog/log"
	"runtime"
	"strings"
	"time"
)

const usage = `Usage: card-dataset-cli [options...]
  -c, --config path to the configuration file (default: ./configs/application.yaml)
  -u, --url dataset download url (only json and zip is supported)
  -f, --file path to local dataset json file, has precedence over the url flag or configuration file
  -h, --help prints help information
`

var file string
var downloadUrl string

func init() {
	logger.SetupConsoleLogger()

	var configPath string

	flag.StringVar(&configPath, "c", "./configs/application.yaml", "path to the configuration file")
	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the configuration file")
	flag.StringVar(&file, "f", "", "path to local dataset json file, has precedence over the url flag or configuration file")
	flag.StringVar(&file, "file", "", "path to local dataset json file, has precedence over the url flag or configuration file")
	flag.StringVar(&downloadUrl, "u", "", "dataset download url (only json and zip is supported)")
	flag.StringVar(&downloadUrl, "url", "", "dataset download url (only json and zip is supported)")
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

	if downloadUrl == "" {
		downloadUrl = cfg.Mtgjson.DownloadURL
	}

	log.Info().Msgf("OS\t\t %s", runtime.GOOS)
	log.Info().Msgf("ARCH\t\t %s", runtime.GOARCH)
	log.Info().Msgf("CPUs\t\t %d", runtime.NumCPU())

	if file == "" {
		log.Info().Msgf("Using dataset from url %s", downloadUrl)
	} else {
		log.Info().Msgf("Using dataset from file %s", file)
	}
}

func main() {
	defer timer.TimeTrack(time.Now(), "import")
	defer stats.LogMemUsage()

	cfg := config.Get()

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

	csService := cardset.NewService(cardset.NewDao(conn))
	cService := card.NewService(card.NewDao(conn))

	imp := mtgjson.NewImporter(csService, cService)

	var report *dataset.Report
	var iErr error
	if file != "" {
		report, iErr = mtgjson.NewFileDataset(imp).Import(strings.NewReader(file))
	} else {
		store, err := storage.NewLocalStorage(cfg.Storage)
		if err != nil {
			log.Error().Err(err).Msg("failed to create local storage")
			return
		}
		allowedTypes := []string{fetch.MimeTypeZip, fetch.MimeTypeJson}
		fetcher := fetch.NewFetcher(cfg.Http, fetch.NewContentTypeValidator(allowedTypes))
		report, iErr = mtgjson.NewDownloadableDataset(imp, fetcher, store).Import(strings.NewReader(downloadUrl))
	}
	if iErr != nil {
		log.Error().Err(iErr).Msg("dataset import failed")
		return
	}
	log.Info().Msgf("Report %#v", report)
}
