package images

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/rs/zerolog/log"
	"io"
	"strings"
)

type ImageResult struct {
	MatchingCardPart string
	MatchingId       int64
	ContentType      string
	File             io.Reader
}

func (img *ImageResult) toCardImage(c *card.Card, lang string) *card.CardImage {
	mimeType := strings.Split(img.ContentType, ";")[0]
	if strings.EqualFold(img.MatchingCardPart, card.PartCard) {
		return &card.CardImage{
			Lang:     lang,
			CardId:   card.NewPrimaryId(img.MatchingId),
			MimeType: mimeType,
		}
	}

	return &card.CardImage{
		Lang:     lang,
		CardId:   c.Id,
		FaceId:   card.NewPrimaryId(img.MatchingId),
		MimeType: mimeType,
	}
}

type Result struct {
	CardImages []*ImageResult
	Missing    int
}

type ImageDownloader interface {
	Download(c *card.Card, lang string) (*Result, error)
}

type Report struct {
	TotalCards int
	Downloaded int
	Missing    int
	Skipped    int
}

type PageConfig struct {
	Page int // starts with 1
	Size int
}
type Images interface {
	Import(PageConfig) (*Report, error)
}

type images struct {
	cardDao    *card.PostgresCardDao
	storage    storage.Storage
	downloader ImageDownloader
	imgReport  *Report
}

func NewImporter(cardDao *card.PostgresCardDao, storage storage.Storage, downloader ImageDownloader) Images {
	return &images{
		cardDao:    cardDao,
		storage:    storage,
		downloader: downloader,
	}
}

func (img *images) Import(pageConfig PageConfig) (*Report, error) {
	page := pageConfig.Page - 1
	if page < 0 {
		page = 0
	}
	if pageConfig.Size < 0 {
		pageConfig.Size = 0
	}

	img.imgReport = &Report{}

	cardCount, err := img.cardDao.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get card count %w", err)
	}

	img.imgReport.TotalCards = cardCount

	cardsPerPage := pageConfig.Size
	maxPages := cardCount / cardsPerPage
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
			for _, lang := range dataset.SupportedLanguages {
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
		img.imgReport.Skipped += 1
		return nil
	}
	result, err := img.downloader.Download(c, lang)
	if err != nil {
		return err
	}

	img.imgReport.Missing += result.Missing

	for _, image := range result.CardImages {
		cardImage := image.toCardImage(c, lang)
		fileName, err := cardImage.BuildFilename()
		if err != nil {
			return fmt.Errorf("failed to build filename %w", err)
		}

		storedFile, err := img.storage.Store(image.File, lang, c.CardSetCode, fileName)
		if err != nil {
			return fmt.Errorf("failed to store card with number %s and set %s %w", c.Number, c.CardSetCode, err)
		}
		cardImage.ImagePath = storedFile.Path

		if err = img.cardDao.AddImage(cardImage); err != nil {
			return fmt.Errorf("failed to add image entry for card name %s, number %s and set %s %w", c.Name, c.Number, c.CardSetCode, err)
		}
		log.Debug().Msgf("stored card image %s for lang %s at %s", c.Name, lang, cardImage.ImagePath)
		img.imgReport.Downloaded += 1
	}

	return nil
}
