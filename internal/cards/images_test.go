package cards_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runner *postgres.DatabaseRunner
var cardDao *cards.PostgresCardDao
var sclient *scryfall.Client

var limitFirstPage20Entries = cards.PageConfig{Page: 1, Size: 20}

func TestImageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	cfg := config.Scryfall{BaseURL: ts.URL, Client: web.Config{}}
	client := web.NewClient(cfg.Client, &http.Client{})
	sclient = scryfall.NewClient(cfg, client, scryfall.DefaultLanguages)

	runner = postgres.NewRunner()
	runner.Run(t, func(t *testing.T) {
		cardDao = cards.NewCardDao(runner.Connection())
		t.Run("import image multiple times", skipAlreadyImportedImages)
		t.Run("import image with phashes", hasPHashes)
		t.Run("import images", importImages)
	})
}

func createCard(t *testing.T, cc ...cards.Card) {
	t.Helper()

	withDefaults := func(c *cards.Card) *cards.Card {
		c.Rarity = "RARE"
		c.Border = "WHITE"
		c.Layout = "NORMAL"

		return c
	}

	cardService := cards.NewCardService(cardDao)
	for _, c := range cc {
		err := cardService.Import(withDefaults(&c))
		require.NoError(t, err, "failed to create card")
	}
}

func hasPHashes(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: t.TempDir()})
	require.NoError(t, err)
	c := cards.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "First",
		Faces: []*cards.Face{
			{
				Name: "First",
			},
		},
	}
	createCard(t, c)

	importer := cards.NewImageImporter(cardDao, localStorage, sclient)
	_, err = importer.Import(limitFirstPage20Entries)
	require.NoError(t, err)

	imgs, err := cardDao.GetImages()
	require.NoError(t, err)

	require.NotZero(t, imgs)
	for _, img := range imgs {
		assert.Greater(t, img.PHash1, uint64(0))
		assert.Greater(t, img.PHash2, uint64(0))
		assert.Greater(t, img.PHash3, uint64(0))
		assert.Greater(t, img.PHash4, uint64(0))
	}
}

func importImages(t *testing.T) {
	cases := []struct {
		name     string
		cards    []cards.Card
		expected cards.ImageReport
	}{
		{
			name: "import second face image",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "multiFace",
					Name:        "FirstFace // SecondFace",
					Faces: []*cards.Face{
						{
							Name: "InvalidName",
						},
						{
							Name: "SecondFace",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   2,
				Missing:    2,
			},
		},
		{
			name: "import first face image",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "multiFace",
					Name:        "FirstFace // SecondFace",
					Faces: []*cards.Face{
						{
							Name: "FirstFace",
						},
						{
							Name: "InvalidName",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   2,
				Missing:    2,
			},
		},
		{
			name: "import all faces images",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "multiFace",
					Name:        "FirstFace // SecondFace",
					Faces: []*cards.Face{
						{
							Name: "FirstFace",
						},
						{
							Name: "SecondFace",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   4,
			},
		},
		{
			name: "no name matches fallback to top card image",
			cards: []cards.Card{
				{

					CardSetCode: "10E",
					Number:      "1",
					Name:        "InvalidName",
					Faces: []*cards.Face{
						{
							Name: "InvalidName",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   2,
			},
		},
		{
			name: "import different card sets",
			cards: []cards.Card{
				{

					CardSetCode: "10E",
					Number:      "1",
					Name:        "First",
					Faces: []*cards.Face{
						{
							Name: "First",
						},
					},
				},
				{
					CardSetCode: "10E",
					Number:      "2",
					Name:        "Second",
					Faces: []*cards.Face{
						{
							Name: "Second",
						},
					},
				},
				{
					CardSetCode: "9E",
					Number:      "3",
					Name:        "Third",
					Faces: []*cards.Face{
						{
							Name: "Third",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 3,
				Imported:   6,
			},
		},
		{
			name: "card exists only with language deu",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "onlydeu",
					Name:        "OnlyDeu",
					Faces: []*cards.Face{
						{
							Name: "OnlyDeu",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   1,
				Missing:    1,
			},
		},
		{
			name: "card exist only with language eng",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "onlyeng",
					Name:        "OnlyEng",
					Faces: []*cards.Face{
						{
							Name: "OnlyEng",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   2,
			},
		},
		{
			name: "card set does not exists",
			cards: []cards.Card{
				{
					CardSetCode: "20E",
					Number:      "1",
					Name:        "First",
					Faces: []*cards.Face{
						{
							Name: "First",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   0,
				Missing:    2,
			},
		},
		{
			name: "card not found",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "99",
					Name:        "First",
					Faces: []*cards.Face{
						{
							Name: "First",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   0,
				Missing:    2,
			},
		},
		{
			name: "image not found",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "imageNotFound",
					Name:        "ImageNotFound",
					Faces: []*cards.Face{
						{
							Name: "ImageNotFound",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   0,
				Missing:    2,
			},
		},
		{
			name: "missing image url",
			cards: []cards.Card{
				{
					CardSetCode: "10E",
					Number:      "noImageUrl",
					Name:        "NoImageUrl",
					Faces: []*cards.Face{
						{
							Name: "NoImageUrl",
						},
					},
				},
			},
			expected: cards.ImageReport{
				TotalCards: 1,
				Imported:   0,
				Missing:    2,
			},
		},
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			dir := t.TempDir()
			localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
			require.NoError(t, err)
			createCard(t, tc.cards...)
			importer := cards.NewImageImporter(cardDao, localStorage, sclient)

			report, err := importer.Import(limitFirstPage20Entries)
			require.NoError(t, err)
			imgCount, err := cardDao.CountImages()
			require.NoError(t, err)

			assert.Equal(t, tc.expected, report)
			assert.Equal(t, tc.expected.Imported, fileCount(t, dir))
			assert.Equal(t, tc.expected.Imported, imgCount)
		})
	}
}

func skipAlreadyImportedImages(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := t.TempDir()
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	require.NoError(t, err)
	createCard(t, cards.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "First",
		Faces: []*cards.Face{
			{
				Name: "First",
			},
		},
	})
	importer := cards.NewImageImporter(cardDao, localStorage, sclient)
	_, err = importer.Import(limitFirstPage20Entries)
	require.NoError(t, err)

	report, err := importer.Import(limitFirstPage20Entries)
	require.NoError(t, err)
	imgCount, err := cardDao.CountImages()
	require.NoError(t, err)

	assert.Equal(t, 2, report.Skipped)
	assert.Equal(t, 2, imgCount)
	assert.Equal(t, 2, fileCount(t, dir))
}

func fileCount(t *testing.T, path string) int {
	t.Helper()

	if len(path) == 0 {
		return 0
	}

	sum := 0
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// ignore directories
		if d.IsDir() {
			return nil
		}

		sum += 1

		return nil
	})
	require.NoErrorf(t, err, "failed to read dir %s", path)

	return sum
}
