package client

import (
	"encoding/json"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

var languages = map[string]string{
	dataset.SupportedLanguages[0]: "de",
	dataset.SupportedLanguages[1]: "en",
}

type ScryfallCard struct {
	Name    string         `json:"name"`
	ImgUris ScyfallImgUris `json:"image_uris"`
	Faces   []ScryfallCard `json:"card_faces"`
}

type ScyfallImgUris struct {
	Normal string `json:"normal"`
}

type MatchingFace struct {
	Url string
	Id  int64
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
		imageUrl := sf.ImgUris.Normal
		if imageUrl == "" {
			log.Warn().Interface("externalCard", sc).Msgf("matching face %s has an empty image url", f.Name)
			continue
		}
		matches = append(matches, &MatchingFace{
			Url: imageUrl,
			Id:  f.Id.Int64,
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
	time.Sleep(time.Millisecond * 25)
	return err
}
