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
var ErrImageBroken = fmt.Errorf("image broken")
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
	Imported   int
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

		filter := Filter{
			SetCode: c.CardSetCode,
			Name:    f.Name,
			Number:  c.Number,
			Lang:    lang,
		}
		cardImg := Image{
			Lang:   lang,
			CardID: c.ID,
			FaceID: f.ID,
		}
		if err := i.addImageData(ctx, &cardImg, filter); err != nil {
			switch {
			case errors.Is(err, ErrCardNotFound):
				log.Warn().Any("filter", filter).Int64("cardID", c.ID.Get()).Int64("faceID", f.ID.Get()).Msg("card not found")

				report.Missing++

				continue
			case errors.Is(err, ErrImageNotFound):
				log.Warn().Any("filter", filter).Int64("cardID", c.ID.Get()).Int64("faceID", f.ID.Get()).Msg("card image not found")

				report.Missing++

				continue
			case errors.Is(err, ErrImageBroken):
				log.Warn().Any("filter", filter).Int64("cardID", c.ID.Get()).Int64("faceID", f.ID.Get()).Msg("card image broken")

				report.Missing++

				continue
			default:
				return err
			}
		}

		if err = i.cardDao.AddImage(ctx, &cardImg); err != nil {
			return fmt.Errorf("failed to add image entry with filter %#v, %w", filter, err)
		}

		log.Debug().Any("filter", filter).Msgf("stored card image at %s", cardImg.ImagePath)

		report.Imported++
	}

	return nil
}

func (i *images) addImageData(ctx context.Context, cardImg *Image, filter Filter) error {
	result, err := i.GetImageWithFallback(ctx, filter, FallbackLang)
	if err != nil {
		return fmt.Errorf("failed to download card image with filter %#v, %w", filter, err)
	}

	cardImg.MimeType = result.MimeType.Raw()

	fileName, err := cardImg.BuildFilename()
	if err != nil {
		return fmt.Errorf("failed to build filename %w", err)
	}

	storedFile, err := i.storer.Store(result.File, filter.Lang, filter.SetCode, fileName)
	if err != nil {
		return fmt.Errorf("failed to store card with filter %#v, %w", filter, err)
	}
	cardImg.ImagePath = storedFile.Path

	fImg, err := os.Open(storedFile.AbsolutePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s, %w", storedFile.AbsolutePath, err)
	}
	defer fImg.Close()

	img, err := jpeg.Decode(fImg)
	if err != nil {
		return fmt.Errorf("failed to decode image %s, %w", storedFile.AbsolutePath, errors.Join(err, ErrImageBroken))
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
