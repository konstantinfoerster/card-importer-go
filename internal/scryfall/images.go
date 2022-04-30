package scryfall

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/images"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall/client"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type processor struct {
	config  config.Scryfall
	client  *client.Client
	storage storage.Storage
}

func NewProcessor(config config.Scryfall, fetcher fetch.Fetcher) images.CardProcessor {
	return &processor{
		config: config,
		client: client.New(fetcher, config),
	}
}

func (p *processor) Process(c *card.Card, lang string) (*images.Result, error) {
	externalCard, err := p.client.GetByCardAndLang(c, lang)
	if err != nil {
		if errors.Is(err, fetch.NotFoundError) {
			log.Warn().Msgf("no scryfall card found with set %s, name %s, number %s and language %s", c.CardSetCode, c.Name, c.Number, lang)
			return &images.Result{}, nil
		}
		return nil, fmt.Errorf("failed to download scryfall card with set %s, name %s, number %s and language %s, reason: %w", c.CardSetCode, c.Name, c.Number, lang, err)
	}

	matches := externalCard.FindMatchingCardParts(c)

	return p.downloadImage(matches, lang)
}

func matchToCardImage(m *client.MatchedPart, lang string) *card.CardImage {
	if m.MatchedType == card.PartCard {
		return &card.CardImage{
			Lang:   lang,
			CardId: card.NewPrimaryId(m.MatchedId),
		}
	}

	return &card.CardImage{
		Lang:   lang,
		FaceId: card.NewPrimaryId(m.MatchedId),
	}
}

func (p *processor) downloadImage(matches []*client.MatchedPart, lang string) (*images.Result, error) {
	var result []func() (*card.CardImage, error)

	for _, m := range matches {
		cardImage := matchToCardImage(m, lang)

		fn := func() (*card.CardImage, error) {
			img, err := p.client.GetImage(m.Url)
			if err != nil {
				if errors.Is(err, fetch.NotFoundError) {
					log.Warn().Interface("url", m.Url).Msg("card image not found")
					return nil, nil
				}
				return nil, fmt.Errorf("failed to download image from %s %w", m.Url, err)
			}
			cardImage.MimeType = img.ContentType
			cardImage.File = img.Body
			return cardImage, nil
		}

		result = append(result, fn)
	}

	return &images.Result{DownloadCard: result}, nil
}
