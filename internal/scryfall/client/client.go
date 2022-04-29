package client

import (
	"encoding/json"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
	"strings"
	"time"
)

var languages = map[string]string{
	api.SupportedLanguages[0]: "de",
	api.SupportedLanguages[1]: "en",
}

type ScryfallCard struct {
	Name    string         `json:"name"`
	ImgUris ScyfallImgUris `json:"image_uris"`
	Faces   []ScryfallCard `json:"card_faces"`
}

type ScyfallImgUris struct {
	Normal string `json:"normal"`
}

type MatchedPart struct {
	Url         string
	CardId      int64
	MatchedType string
	MatchedId   int64
}

func (sc *ScryfallCard) FindMatchingCardParts(c *card.Card) []*MatchedPart {
	// this property is set if there is only one image for the card
	if sc.ImgUris.Normal != "" {
		sf := findMatchingPart([]ScryfallCard{*sc}, c.Name)
		if sf == nil {
			log.Warn().Interface("externalCard", sc).Msgf("no matching entry found for card %s in external card", c.Name)
			return []*MatchedPart{}
		}

		return []*MatchedPart{{
			Url:         sc.ImgUris.Normal,
			CardId:      c.Id.Int64,
			MatchedType: card.PartCard,
			MatchedId:   c.Id.Int64,
		}}
	}

	var matches []*MatchedPart
	for _, f := range c.Faces {
		sf := findMatchingPart(sc.Faces, f.Name)
		if sf == nil {
			log.Warn().Interface("externalCard", sc).Msgf("no matching entry found for face %s in external card", f.Name)
			continue
		}
		imageUrl := sf.ImgUris.Normal
		if imageUrl == "" {
			log.Warn().Interface("externalCard", sc).Msgf("matching face %s has an empty image url", f.Name)
			continue
		}
		matches = append(matches, &MatchedPart{
			Url:         imageUrl,
			CardId:      c.Id.Int64,
			MatchedType: card.PartFace,
			MatchedId:   f.Id.Int64,
		})
	}

	return matches
}

func findMatchingPart(sc []ScryfallCard, term string) *ScryfallCard {
	for _, f := range sc {
		if strings.EqualFold(f.Name, term) {
			return &f
		}
	}

	return nil
}

func New(f fetch.Fetcher, config config.Scryfall) *Client {
	return &Client{
		fetcher: f,
		config:  config,
	}
}

type Client struct {
	fetcher fetch.Fetcher
	config  config.Scryfall
}

func (f *Client) GetByCardAndLang(c *card.Card, lang string) (*ScryfallCard, error) {
	extLang, ok := languages[lang]
	if !ok || extLang == "" {
		return nil, fmt.Errorf("language %s not found in scryfall language mapping %v", lang, languages)
	}

	url := f.config.BuildJsonDownloadURL(c.CardSetCode, c.Number, extLang)
	log.Debug().Msgf("Downloading card metadata from %s", url)

	cardJson, err := f.fetchDelayed(url)
	if err != nil {
		return nil, err
	}
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
	}(cardJson.Body)

	var sc ScryfallCard
	err = json.NewDecoder(cardJson.Body).Decode(&sc)
	if err != nil {
		return nil, fmt.Errorf("failed to decode scryfall card result %w", err)
	}
	return &sc, err
}

func (f *Client) GetImage(url string, afterDownload func(result *fetch.Response) error) error {
	image, err := f.fetchDelayed(url)
	if err != nil {
		return fmt.Errorf("failed to download card image from %s %w", url, err)
	}
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
	}(image.Body)

	err = afterDownload(image)
	return err
}

func (f *Client) fetchDelayed(url string) (*fetch.Response, error) {
	image, err := f.fetcher.Fetch(url)
	time.Sleep(time.Millisecond * 25)
	return image, err
}
