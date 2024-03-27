package scryfall

import (
	"fmt"

	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/images"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall/client"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type downloader struct {
	config config.Scryfall
	client *client.Client
}

func NewDownloader(config config.Scryfall, fetcher fetch.Fetcher) images.ImageDownloader {
	return &downloader{
		config: config,
		client: client.NewClient(fetcher, config),
	}
}

func (d *downloader) Download(c *card.Card, lang string,
	afterDownload func(result *images.ImageResult) error) (*images.DownloadResult, error) {
	externalCard, err := d.client.GetByCardAndLang(c, lang)
	if err != nil {
		if errors.Is(err, fetch.ErrNotFound) {
			log.Warn().Msgf("no scryfall card found with set %s, name %s, number %s and language %s",
				c.CardSetCode, c.Name, c.Number, lang)

			return &images.DownloadResult{Missing: 1}, nil
		}

		return nil, fmt.Errorf("failed to download scryfall card with set %s, name %s, number %s and "+
			"language %s, reason: %w", c.CardSetCode, c.Name, c.Number, lang, err)
	}

	matches := externalCard.FindMatchingCardParts(c)

	var missingImages int
	var downloaded int
	for _, m := range matches {
		err = d.client.GetImage(m.URL, func(resp *fetch.Response) error {
			downloaded++

			return afterDownload(&images.ImageResult{
				MatchingFaceID: m.ID,
				MimeType:       resp.MimeType(),
				File:           resp.Body,
			})
		})
		if err != nil {
			if errors.Is(err, fetch.ErrNotFound) {
				log.Warn().Interface("url", m.URL).Msg("card image not found")
				missingImages++

				continue
			}

			if errors.Is(err, images.ErrBrokenImage) {
				log.Warn().Interface("url", m.URL).Msg("broken image")
				missingImages++

				continue
			}

			return nil, err
		}
	}

	return &images.DownloadResult{Downloaded: downloaded, Missing: missingImages}, nil
}
