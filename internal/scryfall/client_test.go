package scryfall_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindCard(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	cfg := config.Scryfall{BaseURL: ts.URL}
	wclient := web.NewClient(cfg.Client, &http.Client{})
	scryClient := scryfall.NewClient(cfg, wclient, scryfall.DefaultLanguages)

	t.Run("success", func(t *testing.T) {
		expected := &scryfall.Card{
			Name: "First",
			ImgUris: scryfall.ImgURIs{
				Normal: "images/cardImage.jpg",
			},
		}

		scard, err := scryClient.FindCard(t.Context(), "10e", "1", "deu")

		require.NoError(t, err)
		assert.Equal(t, expected, scard)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := scryClient.FindCard(t.Context(), "10e", "1", "eng")

		var apiErr *web.ExternalAPIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	})
}

func TestGetImage(t *testing.T) {
	origImgPath := path.Join("testdata", "images", "cardImage.jpg")
	expectedImg, err := os.ReadFile(origImgPath)
	require.NoError(t, err)

	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	cfg := config.Scryfall{BaseURL: ts.URL}
	wclient := web.NewClient(cfg.Client, &http.Client{})
	scryClient := scryfall.NewClient(cfg, wclient, scryfall.DefaultLanguages)

	t.Run("success", func(t *testing.T) {
		f := cards.Filter{
			SetCode: "10e",
			Number:  "1",
			Lang:    "deu",
			Name:    "First",
		}

		r, err := scryClient.GetImage(t.Context(), f)

		require.NoError(t, err)
		b, err := io.ReadAll(r.File)
		require.NoError(t, err)
		assert.Equal(t, expectedImg, b)
	})
}
