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
	"github.com/konstantinfoerster/card-importer-go/internal/stats"
	"github.com/konstantinfoerster/card-importer-go/internal/timer"
	"github.com/rs/zerolog/log"
	"runtime"
	"strings"
	"time"
)

var file string
var downloadUrl string

func init() {
	logger.SetupConsoleLogger()

	var configPath string

	flag.StringVar(&configPath, "config", "./configs/application.yaml", "path to the config file")
	flag.StringVar(&file, "file", "", "json file to import, has precedence over the url flag or config")
	flag.StringVar(&downloadUrl, "url", "", "download url of the json or zip file")

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
		log.Info().Msgf("Starting with url %s", downloadUrl)
	} else {
		log.Info().Msgf("Starting with file %s", file)
	}
}

func main() {
	defer timer.TimeTrack(time.Now(), "import")

	conn, err := postgres.Connect(context.Background(), config.Get().Database)
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

	var report *api.DatasetReport
	if file != "" {
		report, err = mtgjson.NewFileDataset(imp).Import(strings.NewReader(file))
	} else {
		report, err = mtgjson.NewDownloadableDataset(imp).Import(strings.NewReader(downloadUrl))
	}
	if err != nil {
		log.Error().Err(err).Msg("dataset import failed")
		return
	}
	log.Info().Msgf("Report %#v", report)

	stats.LogMemUsage()
}
