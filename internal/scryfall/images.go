package scryfall

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall/client"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type images struct {
	config    config.Scryfall
	client    *client.Client
	storage   storage.Storage
	cardDao   *card.PostgresCardDao
	imgReport *api.ImageReport
}

func NewImporter(config config.Scryfall, fetcher fetch.Fetcher, storage storage.Storage, cardDao *card.PostgresCardDao) api.Images {
	return &images{
		config:  config,
		client:  client.New(fetcher, config),
		storage: storage,
		cardDao: cardDao,
	}
}

func (img *images) Import() (*api.ImageReport, error) {
	img.imgReport = &api.ImageReport{}

	cardCount, err := img.cardDao.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get card count %w", err)
	}

	cardsPerPage := 20
	maxPages := cardCount / cardsPerPage
	page := 0
	for {
		page = page + 1
		cards, err := img.cardDao.Paged(page, cardsPerPage)
		if err != nil {
			return nil, fmt.Errorf("failed to get card list for page %d and size %d", page, cardsPerPage)
		}
		if len(cards) == 0 {
			break
		}

		log.Info().Msgf("Processing page %d/%d with %d cards", page, maxPages, len(cards))
		for _, c := range cards {
			if err = img.importCardPerLanguage(c, api.SupportedLanguages); err != nil {
				return nil, err
			}
		}
	}

	return img.imgReport, nil
}

func (img *images) importCardPerLanguage(c *card.Card, langs []string) error {
	for _, lang := range langs {
		imgExists, err := img.cardDao.IsImagePresent(c.Id.Get(), lang)
		if err != nil {
			return fmt.Errorf("failed to check if card image already exists for card wtih set %s, "+
				"name %s, number %s and language %s %w", c.CardSetCode, c.Name, c.Number, lang, err)
		}
		if imgExists {
			img.imgReport.ImagesSkipped += 1
			continue
		}

		externalCard, err := img.client.GetByCardAndLang(c, lang)
		if err != nil {
			if errors.Is(err, fetch.NotFoundError) {
				img.imgReport.MissingMetadata += 1
				log.Warn().Msgf("no scryfall card found with set %s, name %s, number %s and language %s", c.CardSetCode, c.Name, c.Number, lang)
				continue
			}
			return fmt.Errorf("failed to download scryfall card with set %s, name %s, number %s and language %s, reason: %w", c.CardSetCode, c.Name, c.Number, lang, err)
		}

		matches := externalCard.FindMatchingCardParts(c)

		if err = img.downloadImage(matches, c, lang); err != nil {
			return err
		}
	}
	return nil
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

func (img *images) downloadImage(matches []*client.MatchedPart, c *card.Card, lang string) error {
	for _, m := range matches {
		cardImage := matchToCardImage(m, lang)

		var storedFile *storage.StoredFile
		storeFn := func(result *fetch.Response) error {
			fileName, err := result.BuildFilename(cardImage.GetFilePrefix())
			if err != nil {
				return fmt.Errorf("failed to build filename %w", err)
			}
			storedFile, err = img.storage.Store(result.Body, lang, c.CardSetCode, fileName)
			if err != nil {
				return fmt.Errorf("failed to store card with number %s and set %s %w", c.Number, c.CardSetCode, err)
			}
			return nil
		}
		err := img.client.GetImage(m.Url, storeFn)
		if err != nil {
			if errors.Is(err, fetch.NotFoundError) {
				log.Warn().Interface("url", m.Url).Msg("card image not found")
				img.imgReport.ImagesMissing += 1
				continue
			}
			return fmt.Errorf("failed to download image for card name %s, number %s and set %s %w", c.Name, c.Number, c.CardSetCode, err)
		}
		cardImage.ImagePath = storedFile.Path
		if err = img.cardDao.AddImage(cardImage); err != nil {
			return fmt.Errorf("failed to add image entry for card name %s, number %s and set %s %w", c.Name, c.Number, c.CardSetCode, err)
		}

		log.Debug().Msgf("stored card image %s for lang %s at %s", c.Name, lang, storedFile.Path)
		img.imgReport.ImagesDownloaded += 1
	}

	return nil
}
