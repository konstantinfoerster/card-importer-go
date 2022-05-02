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

func (d *downloader) Download(c *card.Card, lang string) (*images.Result, error) {
	externalCard, err := d.client.GetByCardAndLang(c, lang)
	if err != nil {
		if errors.Is(err, fetch.NotFoundError) {
			log.Warn().Msgf("no scryfall card found with set %s, name %s, number %s and language %s", c.CardSetCode, c.Name, c.Number, lang)
			return &images.Result{Missing: 1}, nil
		}
		return nil, fmt.Errorf("failed to download scryfall card with set %s, name %s, number %s and language %s, reason: %w", c.CardSetCode, c.Name, c.Number, lang, err)
	}

	matches := externalCard.FindMatchingCardParts(c)

	return d.downloadImage(matches)
}

func (d *downloader) downloadImage(matches []*client.MatchedPart) (*images.Result, error) {
	var result []*images.ImageResult

	var missingImages int
	for _, m := range matches {
		img, err := d.client.GetImage(m.Url)
		if err != nil {
			if errors.Is(err, fetch.NotFoundError) {
				log.Warn().Interface("url", m.Url).Msg("card image not found")
				missingImages += 1
				continue
			}
			return nil, fmt.Errorf("failed to download image from %s %w", m.Url, err)
		}

		cardImage := &images.ImageResult{
			MatchingId:       m.MatchedId,
			MatchingCardPart: m.MatchedType,
			ContentType:      img.ContentType,
			File:             img.Body,
		}

		result = append(result, cardImage)
	}

	return &images.Result{CardImages: result, Missing: missingImages}, nil
}
