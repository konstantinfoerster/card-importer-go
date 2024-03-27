package images

import (
	"fmt"
	"image/jpeg"
	"io"
	"os"

	"github.com/anthonynsimon/bild/transform"
	"github.com/corona10/goimagehash"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/rs/zerolog/log"
)

var ErrBrokenImage = fmt.Errorf("broken image")

type ImageResult struct {
	MatchingFaceID int64
	MimeType       fetch.MimeType
	File           io.Reader
}

func (img *ImageResult) toCardImage(c *card.Card, lang string) *card.Image {
	return &card.Image{
		Lang:     lang,
		CardID:   c.ID,
		FaceID:   card.NewPrimaryID(img.MatchingFaceID),
		MimeType: img.MimeType.Raw(),
	}
}

type DownloadResult struct {
	Downloaded int
	Missing    int
}

type ImageDownloader interface {
	Download(c *card.Card, lang string, afterDownload func(result *ImageResult) error) (*DownloadResult, error)
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
		page++
		cards, err := img.cardDao.Paged(page, cardsPerPage)
		if err != nil {
			return nil, fmt.Errorf("failed to get card list for page %d and size %d. %w", page, cardsPerPage, err)
		}
		if len(cards) == 0 {
			break
		}

		log.Info().Msgf("Processing page %d/%d with %d cards", page, maxPages, len(cards))
		for _, c := range cards {
			for _, lang := range dataset.GetSupportedLanguages() {
				if err = img.importCard(c, lang); err != nil {
					return nil, err
				}
			}
		}
	}

	return img.imgReport, nil
}

func (i *images) importCard(c *card.Card, lang string) error {
	imgExists, err := i.cardDao.IsImagePresent(c.ID.Get(), lang)
	if err != nil {
		return fmt.Errorf("failed to check if card image already exists for card with set %s, "+
			"name %s, number %s and language %s %w", c.CardSetCode, c.Name, c.Number, lang, err)
	}
	if imgExists {
		i.imgReport.Skipped++

		return nil
	}

	afterDownload := func(result *ImageResult) error {
		cardImage := result.toCardImage(c, lang)
		fileName, err := cardImage.BuildFilename()
		if err != nil {
			return fmt.Errorf("failed to build filename %w", err)
		}

		storedFile, err := i.storage.Store(result.File, lang, c.CardSetCode, fileName)
		if err != nil {
			return fmt.Errorf("failed to store card with number %s and set %s %w", c.Number, c.CardSetCode, err)
		}
		cardImage.ImagePath = storedFile.Path

		f, err := os.Open(storedFile.AbsolutePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s, %w", storedFile.AbsolutePath, err)
		}

		img, err := jpeg.Decode(f)
		if err != nil {
			return fmt.Errorf("%w, failed to decode image %s, %w", ErrBrokenImage, storedFile.AbsolutePath, err)
		}

		imgHash, err := goimagehash.PerceptionHash(img)
		if err != nil {
			return fmt.Errorf("failed to create phash from %s, %w", storedFile.AbsolutePath, err)
		}
		cardImage.PHash = imgHash.GetHash()

		rotated := transform.Rotate(img, 180, nil)
		rotatedImgHash, err := goimagehash.PerceptionHash(rotated)
		if err != nil {
			return fmt.Errorf("failed to create rotated phash from %s, %w", storedFile.AbsolutePath, err)
		}
		cardImage.PHashRotated = rotatedImgHash.GetHash()

		if err = i.cardDao.AddImage(cardImage); err != nil {
			return fmt.Errorf("failed to add image entry for card name %s, number %s and set %s %w",
				c.Name, c.Number, c.CardSetCode, err)
		}
		log.Debug().Msgf("stored card image %s for lang %s at %s", c.Name, lang, cardImage.ImagePath)

		return nil
	}

	result, err := i.downloader.Download(c, lang, afterDownload)
	if err != nil {
		return err
	}

	i.imgReport.Missing += result.Missing
	i.imgReport.Downloaded += result.Downloaded

	return nil
}
