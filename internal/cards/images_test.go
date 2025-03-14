package cards_test

import (
	"net/http"
	"net/http/httptest"
	"os"
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
		t.Run("import images for different sets", importDifferentSets)
		t.Run("import image multiple times", importSameImageMultipleTimes)
		t.Run("import image with phashes", hasPHashes)
		t.Run("import even when face name does not match", importWithoutFaceMatch)
		t.Run("no card is matching", noCardMatches)
		t.Run("skips missing languages", importNextLanguageOnMissingCard)
		t.Run("skips missing faces", importMultiFaces)
		t.Run("error cases", testErrorCases)
	})
}

func createCard(t *testing.T, c *cards.Card) {
	t.Helper()

	withDefaults := func(c *cards.Card) *cards.Card {
		c.Rarity = "RARE"
		c.Border = "WHITE"
		c.Layout = "NORMAL"

		return c
	}

	cardService := cards.NewCardService(cardDao)
	err := cardService.Import(withDefaults(c))

	require.NoError(t, err, "failed to create card")
}

func noCardMatches(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: t.TempDir()})
	require.NoError(t, err)

	createCard(t, &cards.Card{
		CardSetCode: "20E",
		Number:      "1",
		Name:        "First",
		Faces: []*cards.Face{
			{
				Name: "First",
			},
		},
	})

	importer := cards.NewImageImporter(cardDao, localStorage, sclient)
	report, err := importer.Import(limitFirstPage20Entries)
	require.NoError(t, err)

	imgCount, err := cardDao.CountImages()
	require.NoError(t, err)

	assert.Equal(t, 0, report.Downloaded)
	assert.Equal(t, 0, imgCount)
}

func importDifferentSets(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := t.TempDir()
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	require.NoError(t, err)

	createCard(t, &cards.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "First",
		Faces: []*cards.Face{
			{
				Name:   "First",
				Colors: cards.NewColors([]string{"W", "B", "R"}),
			},
		},
	})
	createCard(t, &cards.Card{
		CardSetCode: "10E",
		Number:      "2",
		Name:        "Second",
		Faces: []*cards.Face{
			{
				Name:   "Second",
				Colors: cards.NewColors([]string{"W"}),
			},
		},
	})
	createCard(t, &cards.Card{
		CardSetCode: "9E",
		Number:      "3",
		Name:        "Third",
		Faces: []*cards.Face{
			{
				Name: "Third",
			},
		},
	})

	importer := cards.NewImageImporter(cardDao, localStorage, sclient)
	report, err := importer.Import(limitFirstPage20Entries)
	require.NoError(t, err)

	imgCount, err := cardDao.CountImages()
	require.NoError(t, err)

	assert.Equal(t, 6, report.Downloaded)
	assert.Equal(t, 6, imgCount)
	assertFileCount(t, filepath.Join(dir, "deu", "10E"), 2)
	assertFileCount(t, filepath.Join(dir, "eng", "10E"), 2)
	assertFileCount(t, filepath.Join(dir, "eng", "9E"), 1)
	assertFileCount(t, filepath.Join(dir, "deu", "9E"), 1)
}

func hasPHashes(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: t.TempDir()})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	c := &cards.Card{
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
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	img, err := cardDao.GetImage(c.ID.Int64, "eng")
	if err != nil {
		t.Fatalf("find card failed %v", err)
	}

	assert.Greater(t, img.PHash1, uint64(0))
	assert.Greater(t, img.PHash2, uint64(0))
	assert.Greater(t, img.PHash3, uint64(0))
	assert.Greater(t, img.PHash4, uint64(0))
}

func importWithoutFaceMatch(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := t.TempDir()
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	createCard(t, &cards.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "DoesNotMatch",
		Faces: []*cards.Face{
			{
				Name: "DoesNotMatch",
			},
		},
	})

	importer := cards.NewImageImporter(cardDao, localStorage, sclient)
	report, err := importer.Import(limitFirstPage20Entries)
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	imgCount, err := cardDao.CountImages()
	if err != nil {
		t.Fatalf("image count failed %v", err)
	}

	assert.Equal(t, 2, report.Downloaded)
	assert.Equal(t, 2, imgCount)
	assertFileCount(t, filepath.Join(dir, "deu", "10E"), 1)
	assertFileCount(t, filepath.Join(dir, "eng", "10E"), 1)
}

func importNextLanguageOnMissingCard(t *testing.T) {
	cases := []struct {
		name             string
		cards            cards.Card
		lang             string
		expectedImgCount int
		expectedFiles    int
	}{
		{
			name: "card exists only with language deu",
			cards: cards.Card{
				CardSetCode: "10E",
				Number:      "onlydeu",
				Name:        "OnlyDeu",
				Faces: []*cards.Face{
					{
						Name: "OnlyDeu",
					},
				},
			},
			lang:             "deu",
			expectedImgCount: 1,
			expectedFiles:    1,
		},
		{
			name: "card exist only with language eng",
			cards: cards.Card{
				CardSetCode: "10E",
				Number:      "onlyeng",
				Name:        "OnlyEng",
				Faces: []*cards.Face{
					{
						Name: "OnlyEng",
					},
				},
			},
			lang: "eng",
			// deu will fallback to eng and import successfully
			expectedImgCount: 2,
			// but image will only be one
			expectedFiles: 1,
		},
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			dir := t.TempDir()
			localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir, Mode: config.CREATE})
			require.NoError(t, err)
			createCard(t, &tc.cards)

			importer := cards.NewImageImporter(cardDao, localStorage, sclient)
			report, err := importer.Import(limitFirstPage20Entries)
			require.NoError(t, err)

			imgCount, err := cardDao.CountImages()
			require.NoError(t, err)

			assert.Equal(t, tc.expectedImgCount, report.Downloaded)
			assert.Equal(t, tc.expectedImgCount, imgCount)
			p := filepath.Join(dir, tc.lang, tc.cards.CardSetCode)
			assertFileCount(t, p, tc.expectedFiles)
		})
	}
}

func importMultiFaces(t *testing.T) {
	cases := []struct {
		name    string
		fixture cards.Card
		want    int
	}{
		{
			name: "FirstFaceDoesNotMatch",
			fixture: cards.Card{
				CardSetCode: "10E",
				Number:      "multiFace",
				Name:        "DoesNotMatch // First",
				Faces: []*cards.Face{
					{
						Name: "DoesNotMatch",
					},
					{
						Name: "SecondFace",
					},
				},
			},
			want: 1,
		},
		{
			name: "SecondFaceDoesNotMatch",
			fixture: cards.Card{
				CardSetCode: "10E",
				Number:      "multiFace",
				Name:        "First // DoesNotMatch",
				Faces: []*cards.Face{
					{
						Name: "FirstFace",
					},
					{
						Name: "DoesNotMatch",
					},
				},
			},
			want: 1,
		},
		{
			name: "BothFacesMatch",
			fixture: cards.Card{
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
			want: 2,
		},
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			dir := t.TempDir()
			localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
			if err != nil {
				t.Fatalf("failed to create local storage %v", err)
			}
			createCard(t, &tc.fixture)

			importer := cards.NewImageImporter(cardDao, localStorage, sclient)
			report, err := importer.Import(limitFirstPage20Entries)
			if err != nil {
				t.Fatalf("import failed %v", err)
			}
			imgCount, err := cardDao.CountImages()
			if err != nil {
				t.Fatalf("image count failed %v", err)
			}

			assert.Equal(t, tc.want*2, report.Downloaded)
			assert.Equal(t, tc.want*2, imgCount)
			assertFileCount(t, filepath.Join(dir, "deu", "10E"), tc.want)
			assertFileCount(t, filepath.Join(dir, "eng", "10E"), tc.want)
		})
	}
}

func importSameImageMultipleTimes(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := t.TempDir()
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	createCard(t, &cards.Card{
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
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	report, err := importer.Import(limitFirstPage20Entries)
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	imgCount, err := cardDao.CountImages()
	if err != nil {
		t.Fatalf("image count failed %v", err)
	}

	assert.Equal(t, 2, report.Skipped)
	assert.Equal(t, 2, imgCount)
	assertFileCount(t, filepath.Join(dir, "deu", "10E"), 1)
	assertFileCount(t, filepath.Join(dir, "eng", "10E"), 1)
}

func testErrorCases(t *testing.T) {
	dir := t.TempDir()
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}

	cases := []struct {
		name string
		card cards.Card
	}{
		{
			name: "ExternalCardNotFound",
			card: cards.Card{
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
		{
			name: "NoImageFound",
			card: cards.Card{
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
		{
			name: "NoImageUrlAttributeSet",
			card: cards.Card{
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
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			createCard(t, &tc.card)

			importer := cards.NewImageImporter(cardDao, localStorage, sclient)

			report, err := importer.Import(limitFirstPage20Entries)
			require.NoError(t, err)

			imgCount, err := cardDao.CountImages()
			require.NoError(t, err)

			assert.Equal(t, 0, report.Downloaded)
			assert.Equal(t, 0, imgCount)
			assertFileCount(t, dir, 0)
		})
	}
}

func assertFileCount(t *testing.T, path string, expectedCount int) {
	t.Helper()

	files, err := os.ReadDir(path)
	require.NoErrorf(t, err, "failed to read dir %s", path)
	assert.Equal(t, expectedCount, len(files))
}
