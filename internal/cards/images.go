package cards

import (
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"os"
	"strings"

	"github.com/corona10/goimagehash"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/rs/zerolog/log"
)

var ErrImageNotFound = fmt.Errorf("image not found")
var ErrCardNotFound = fmt.Errorf("card not found")

type ImageResult struct {
	File     io.ReadCloser
	MimeType web.MimeType
}

func NewFilter(setCode, name, number, lang string) (Filter, error) {
	var err error
	if strings.TrimSpace(setCode) == "" {
		err = errors.Join(err, errors.New("missing set-code"))
	}
	if strings.TrimSpace(name) == "" {
		err = errors.Join(err, errors.New("missing name"))
	}
	if strings.TrimSpace(number) == "" {
		err = errors.Join(err, errors.New("missing number"))
	}
	if err != nil {
		return Filter{}, err
	}

	if strings.TrimSpace(lang) == "" {
		lang = DefaultLang
	}

	return Filter{
		SetCode: setCode,
		Name:    name,
		Number:  number,
		Lang:    lang,
	}, nil
}

type Filter struct {
	SetCode string
	Name    string
	Number  string
	Lang    string
}

type ImageDownloader interface {
	GetImage(ctx context.Context, f Filter) (*ImageResult, error)
}

type ImageReport struct {
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
	Import(PageConfig) (ImageReport, error)
}

type images struct {
	cardDao    *PostgresCardDao
	storer     storage.Storer
	downloader ImageDownloader
}

func NewImageImporter(cardDao *PostgresCardDao, storer storage.Storer, downloader ImageDownloader) Images {
	return &images{
		cardDao:    cardDao,
		storer:     storer,
		downloader: downloader,
	}
}

func (i *images) Import(pageConfig PageConfig) (ImageReport, error) {
	ctx := context.Background()

	page := max(pageConfig.Page-1, 0)
	pageSize := max(pageConfig.Size, 0)
	report := ImageReport{}

	cardCount, err := i.cardDao.Count()
	if err != nil {
		return ImageReport{}, fmt.Errorf("failed to get card count %w", err)
	}

	report.TotalCards = cardCount

	maxPages := cardCount / pageSize
	for {
		page++
		cards, err := i.cardDao.Paged(page, pageSize)
		if err != nil {
			return ImageReport{}, fmt.Errorf("failed to get card list for page %d and size %d. %w", page, pageSize, err)
		}
		if len(cards) == 0 {
			break
		}

		log.Info().Msgf("Processing page %d/%d with %d cards", page, maxPages, len(cards))
		for _, c := range cards {
			for _, lang := range GetSupportedLanguages() {
				if err = i.importCard(ctx, c, lang, &report); err != nil {
					return ImageReport{}, err
				}
			}
		}
	}

	return report, nil
}

func (i *images) importCard(ctx context.Context, c Card, lang string, report *ImageReport) error {
	for _, f := range c.Faces {
		imgExists, err := i.cardDao.IsImagePresent(ctx, f.ID.Get(), lang)
		if err != nil {
			return fmt.Errorf("failed to check if card image already exists for card face with set %s, "+
				"name %s, number %s and language %s, %w", c.CardSetCode, f.Name, c.Number, lang, err)
		}
		if imgExists {
			report.Skipped++

			continue
		}

		if err := i.importFace(ctx, c, f, lang, report); err != nil {
			return err
		}
	}

	return nil
}

func (i *images) importFace(ctx context.Context, c Card, f *Face, lang string, report *ImageReport) error {
	filter := Filter{
		SetCode: c.CardSetCode,
		Name:    f.Name,
		Number:  c.Number,
		Lang:    lang,
	}
	result, err := i.GetImageWithFallback(ctx, filter, FallbackLang)
	if err != nil {
		if errors.Is(err, ErrCardNotFound) {
			log.Warn().Any("filter", filter).Int64("cardID", c.ID.Get()).Int64("faceID", f.ID.Get()).Msg("card not found")

			report.Missing++

			return nil
		} else if errors.Is(err, ErrImageNotFound) {
			log.Warn().Any("filter", filter).Int64("cardID", c.ID.Get()).Int64("faceID", f.ID.Get()).Msg("card image not found")

			report.Missing++

			return nil
		}

		return fmt.Errorf("failed to download card image with filter %#v, %w", f, err)
	}

	cardImg := &Image{
		Lang:     lang,
		CardID:   c.ID,
		FaceID:   f.ID,
		MimeType: result.MimeType.Raw(),
	}
	fileName, err := cardImg.BuildFilename()
	if err != nil {
		return fmt.Errorf("failed to build filename %w", err)
	}

	storedFile, err := i.storer.Store(result.File, lang, c.CardSetCode, fileName)
	if err != nil {
		return fmt.Errorf("failed to store card with number %s and set %s, %w", c.Number, c.CardSetCode, err)
	}
	cardImg.ImagePath = storedFile.Path

	fImg, err := os.Open(storedFile.AbsolutePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s, %w", storedFile.AbsolutePath, err)
	}
	defer fImg.Close()

	img, err := jpeg.Decode(fImg)
	if err != nil {
		return fmt.Errorf("failed to decode image %s, %w", storedFile.AbsolutePath, err)
	}

	imgWidth := 16
	imgHeight := imgWidth
	imgPHash, err := goimagehash.ExtPerceptionHash(img, imgWidth, imgHeight)
	if err != nil {
		return fmt.Errorf("failed to create phash from %s, %w", cardImg.ImagePath, err)
	}
	cardImg.PHash1 = imgPHash.GetHash()[0]
	cardImg.PHash2 = imgPHash.GetHash()[1]
	cardImg.PHash3 = imgPHash.GetHash()[2]
	cardImg.PHash4 = imgPHash.GetHash()[3]

	if err = i.cardDao.AddImage(ctx, cardImg); err != nil {
		return fmt.Errorf("failed to add image entry for card name %s, number %s and set %s %w",
			c.Name, c.Number, c.CardSetCode, err)
	}
	log.Debug().Msgf("stored card image %s for lang %s at %s", c.Name, lang, cardImg.ImagePath)

	report.Downloaded++

	return nil
}

func (i *images) GetImageWithFallback(ctx context.Context, filter Filter, fallbackLang string) (*ImageResult, error) {
	result, err := i.downloader.GetImage(ctx, filter)
	if err != nil {
		if errors.Is(err, ErrImageNotFound) && filter.Lang != fallbackLang {
			// try to get image for another language
			filter.Lang = fallbackLang

			return i.downloader.GetImage(ctx, filter)
		}

		return nil, err
	}

	return result, nil
}
