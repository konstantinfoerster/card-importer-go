package mtgjson

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/aio"
	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/rs/zerolog/log"
)

type FileLoader struct {
	dataset cards.Dataset
	cfg     config.Mtgjson
	client  web.Client
	store   storage.Storer
}

func NewLoader(dataset cards.Dataset, cfg config.Mtgjson, wclient web.Client, store storage.Storer) *FileLoader {
	return &FileLoader{
		dataset: dataset,
		cfg:     cfg,
		client:  wclient,
		store:   store,
	}
}

func (l *FileLoader) Load(source *url.URL) (*cards.Report, error) {
	if source.Scheme == "" {
		filePath := filepath.Clean(source.String())
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s %w", filePath, err)
		}
		defer aio.Close(f)

		return l.dataset.Import(f)
	}

	ctx := context.Background()
	opts := web.NewGetOpts().WithExpectedCodes(200)
	resp, err := l.client.Get(ctx, source.String(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to download dataset from %s due to %w", source, err)
	}
	defer aio.Close(resp.Body)

	filename, err := resp.MimeType.BuildFilename(fmt.Sprintf("%d", time.Now().UnixMilli()))
	if err != nil {
		return nil, err
	}

	sFile, err := l.store.Store(resp.Body, "downloads", filename)
	if err != nil {
		return nil, err
	}

	fileToImport, err := extract(sFile.AbsolutePath, resp.MimeType)
	if err != nil {
		return nil, err
	}

	fileToImport = filepath.Clean(fileToImport)
	f, err := os.Open(fileToImport)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s %w", fileToImport, err)
	}
	defer aio.Close(f)

	return l.dataset.Import(f)
}

func extract(file string, m web.MimeType) (string, error) {
	if !m.IsZip() {
		return file, nil
	}
	var err error
	defer func(name string) {
		rErr := os.Remove(name)
		if rErr != nil {
			// report remove errors
			if err == nil {
				err = rErr
			} else {
				err = errors.Join(err, rErr)
			}
		} else {
			log.Info().Msgf("Delete zip file %s", name)
		}
	}(file)

	dest := filepath.Dir(file)
	log.Info().Msgf("Unzipping %s to %s", file, dest)
	files, err := unzip(file, dest)
	if err != nil {
		return "", err
	}
	log.Info().Msgf("Unzip finished with files %v", files)

	if len(files) != 1 {
		return "", fmt.Errorf("unexpected file count inside zip file, expected 1 but found %d", len(files))
	}

	return files[0], err
}
