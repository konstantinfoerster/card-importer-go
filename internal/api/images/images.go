package images

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
)

type Report struct {
	ImagesDownloaded int
	ImagesMissing    int
	ImagesSkipped    int
}

type PageConfig struct {
	Page int
	Size int
}
type Images interface {
	Import(PageConfig) (*Report, error)
}

type Result struct {
	DownloadCard []func() (*card.CardImage, error)
}

type CardProcessor interface {
	Process(c *card.Card, lang string) (*Result, error)
}

type images struct {
	cardDao   *card.PostgresCardDao
	storage   storage.Storage
	processor CardProcessor
	imgReport *Report
}

func NewImporter(cardDao *card.PostgresCardDao, storage storage.Storage, processor CardProcessor) Images {
	return &images{
		cardDao:   cardDao,
		storage:   storage,
		processor: processor,
	}
}

func (img *images) Import(pageConfig PageConfig) (*Report, error) {
	if pageConfig.Page <= 0 {
		pageConfig.Page = 1
	}
	if pageConfig.Size < 0 {
		pageConfig.Size = 0
	}

	img.imgReport = &Report{}

	cardCount, err := img.cardDao.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get card count %w", err)
	}

	cardsPerPage := pageConfig.Size
	maxPages := cardCount / cardsPerPage
	page := pageConfig.Page - 1
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
			for _, lang := range api.SupportedLanguages {
				if err = img.importCard(c, lang); err != nil {
					return nil, err
				}
			}
		}
	}

	return img.imgReport, nil
}

func (img *images) importCard(c *card.Card, lang string) error {
	imgExists, err := img.cardDao.IsImagePresent(c.Id.Get(), lang)
	if err != nil {
		return fmt.Errorf("failed to check if card image already exists for card wtih set %s, "+
			"name %s, number %s and language %s %w", c.CardSetCode, c.Name, c.Number, lang, err)
	}
	if imgExists {
		img.imgReport.ImagesSkipped += 1
		return nil
	}
	result, err := img.processor.Process(c, lang)
	if err != nil {
		return err
	}

	for _, downloadCardFn := range result.DownloadCard {
		cardImage, err := downloadCardFn()
		if err != nil {
			return err
		}
		if cardImage == nil {
			img.imgReport.ImagesMissing += 1
			continue
		}
		if err := img.storeCard(cardImage, c, lang); err != nil {
			return err
		}
		if err = img.cardDao.AddImage(cardImage); err != nil {
			return fmt.Errorf("failed to add image entry for card name %s, number %s and set %s %w", c.Name, c.Number, c.CardSetCode, err)
		}
		log.Debug().Msgf("stored card image %s for lang %s at %s", c.Name, lang, cardImage.ImagePath)
		img.imgReport.ImagesDownloaded += 1
	}

	return nil
}

func (img *images) storeCard(cardImage *card.CardImage, c *card.Card, lang string) error {
	var err error
	defer func(toClose io.ReadCloser) {
		cErr := toClose.Close()
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Wrap(err, cErr.Error())
			}
		}
	}(cardImage.File)

	fileName, err := cardImage.BuildFilename()
	if err != nil {
		return fmt.Errorf("failed to build filename %w", err)
	}

	storedFile, err := img.storage.Store(cardImage.File, lang, c.CardSetCode, fileName)
	if err != nil {
		return fmt.Errorf("failed to store card with number %s and set %s %w", c.Number, c.CardSetCode, err)
	}
	cardImage.ImagePath = storedFile.Path

	return nil
}
