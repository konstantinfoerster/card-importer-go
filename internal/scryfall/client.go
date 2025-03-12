package scryfall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
)

var DefaultLanguages = cards.NewLanguageMapper(map[string]string{
	cards.GetSupportedLanguages()[0]: "de",
	cards.GetSupportedLanguages()[1]: "en",
})

func NewClient(cfg config.Scryfall, wclient web.Client, languages cards.LanguageMapper) *Client {
	return &Client{
		cfg:       cfg,
		wclient:   wclient,
		languages: languages,
	}
}

type Client struct {
	cfg       config.Scryfall
	wclient   web.Client
	languages cards.LanguageMapper
}

func (c *Client) FindCard(ctx context.Context, setCode, number, lang string) (*Card, error) {
	targetLang, err := c.languages.Get(lang)
	if err != nil {
		return nil, fmt.Errorf("language %s not found due to %w", lang, err)
	}

	url, err := c.cfg.EnsureBaseURL(path.Join("cards", strings.ToLower(setCode), strings.ToUpper(number), targetLang))
	if err != nil {
		return nil, fmt.Errorf("failed to create get card url due to invalid url due to %w", err)
	}
	url += "?format=json&version=normal"

	opts := web.NewGetOpts().
		WithHeader(web.HeaderAccept, web.MimeTypeJSON).
		WithExpectedCodes(200)
	resp, err := c.wclient.Get(ctx, url, opts)
	if err != nil {
		if web.IsStatusCode(err, http.StatusNotFound) {
			err = errors.Join(err, cards.ErrCardNotFound)
		}

		return nil, fmt.Errorf("failed to find card %s due to %w", url, err)
	}

	var sc Card
	if err := json.NewDecoder(resp.Body).Decode(&sc); err != nil {
		return nil, fmt.Errorf("failed to decode scryfall card due to %w", err)
	}

	return &sc, nil
}

func (c *Client) GetImage(ctx context.Context, f cards.Filter) (*cards.ImageResult, error) {
	sCard, err := c.FindCard(ctx, f.SetCode, f.Number, f.Lang)
	if err != nil {
		return nil, errors.Join(cards.ErrImageNotFound, err)
	}

	imgURLRaw := sCard.FindURL(f.Name)
	if imgURLRaw == "" {
		return nil, fmt.Errorf("no matching scryfall card image with set %s, name %s, number %s and "+
			"language %s found due to %w", f.SetCode, f.Name, f.Number, f.Lang, cards.ErrImageNotFound)
	}
	imgURL, err := c.cfg.EnsureBaseURL(imgURLRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid scryfall card image url %s, %w", imgURL, errors.Join(err, cards.ErrImageNotFound))
	}

	opts := web.NewGetOpts().
		WithHeader(web.HeaderAccept, web.MimeTypeJpeg).
		WithExpectedCodes(200)
	resp, err := c.wclient.Get(ctx, imgURL, opts)
	if err != nil {
		if web.IsStatusCode(err, http.StatusNotFound) {
			err = errors.Join(err, cards.ErrImageNotFound)
		}

		return nil, fmt.Errorf("failed to get image from %s due to %w", imgURL, err)
	}

	return &cards.ImageResult{
		MimeType: resp.MimeType,
		File:     resp.Body,
	}, nil
}
