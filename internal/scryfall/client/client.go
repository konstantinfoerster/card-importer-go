package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/rs/zerolog/log"
)

type ScryfallCard struct {
	Name string `json:"name"`
	// external-struct
	ImgUris ScyfallImgURIs `json:"image_uris"`
	// external-struct
	Faces []ScryfallCard `json:"card_faces"`
}

type ScyfallImgURIs struct {
	Normal string `json:"normal"`
}

type MatchingFace struct {
	URL string
	ID  int64
}

func (sc *ScryfallCard) FindMatchingCardParts(c *card.Card) []*MatchingFace {
	possibleCards := []ScryfallCard{*sc}
	possibleCards = append(possibleCards, sc.Faces...)

	var matches []*MatchingFace
	for _, f := range c.Faces {
		sf := findMatchingPart(possibleCards, f.Name)
		if sf == nil {
			log.Warn().Interface("externalCard", sc).Msgf("no matching entry found for face %s in external card", f.Name)

			continue
		}
		imageURL := sf.ImgUris.Normal
		if imageURL == "" {
			log.Warn().Interface("externalCard", sc).Msgf("matching face %s has an empty image url", f.Name)

			continue
		}
		matches = append(matches, &MatchingFace{
			URL: imageURL,
			ID:  f.ID.Int64,
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

func NewClient(f fetch.Fetcher, config config.Scryfall) *Client {
	return &Client{
		fetcher: f,
		config:  config,
		languages: dataset.NewLanguageMapper(
			map[string]string{
				dataset.GetSupportedLanguages()[0]: "de",
				dataset.GetSupportedLanguages()[1]: "en",
			},
		),
	}
}

type Client struct {
	fetcher   fetch.Fetcher
	config    config.Scryfall
	languages dataset.LanguageMapper
}

func (f *Client) GetByCardAndLang(c *card.Card, lang string) (*ScryfallCard, error) {
	extLang, err := f.languages.Get(lang)
	if err != nil {
		return nil, fmt.Errorf("language %s not found %w", lang, err)
	}

	url := f.config.BuildJSONDownloadURL(c.CardSetCode, c.Number, extLang)
	log.Debug().Msgf("Downloading card metadata from %s", url)

	var sc ScryfallCard
	decodeFn := func(resp *fetch.Response) error {
		err := json.NewDecoder(resp.Body).Decode(&sc)
		if err != nil {
			return fmt.Errorf("failed to decode scryfall card result %w", err)
		}

		return nil
	}

	if err := f.fetchDelayed(url, decodeFn); err != nil {
		return nil, err
	}

	return &sc, nil
}

func (f *Client) GetImage(url string, handleResponse func(resp *fetch.Response) error) error {
	err := f.fetchDelayed(url, handleResponse)
	if err != nil {
		return fmt.Errorf("failed to download card image from %s %w", url, err)
	}

	return nil
}

func (f *Client) fetchDelayed(url string, handleResponse func(resp *fetch.Response) error) error {
	err := f.fetcher.Fetch(url, handleResponse)
	time.Sleep(time.Millisecond * 50) // TODO make this configurable

	return err
}
