package scryfall_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/images"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logger.SetupConsoleLogger()
	err := logger.SetLogLevel("warn")
	if err != nil {
		fmt.Printf("Failed to set log level %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type MockFetcher struct {
	FakeFetch func(url string, handleResponse func(result *fetch.Response) error) error
}

func (p *MockFetcher) Fetch(url string, handleResponse func(resp *fetch.Response) error) error {
	return p.FakeFetch(url, handleResponse)
}

var runner *postgres.DatabaseRunner
var fetcher fetch.Fetcher
var cfg config.Scryfall
var cardDao *card.PostgresCardDao

var limitFirstPage20Entries = images.PageConfig{Page: 1, Size: 20}

func TestImageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	cfg = config.Scryfall{DownloadURL: "http://localhost/{code}/{number}/{lang}.json"}
	fetcher = testdataFileFetcher(t)

	runner = postgres.NewRunner()
	runner.Run(t, func(t *testing.T) {
		cardDao = card.NewDao(runner.Connection())
		t.Run("import images for different sets", importDifferentSets)
		t.Run("import image multiple times", importSameImageMultipleTimes)
		t.Run("import image with phashes", hasPHashes)
		t.Run("ignores card name cases", ignoresCardNamesCases)
		t.Run("no card is matching", noCardMatches)
		t.Run("skips missing languages", importNextLanguageOnMissingCard)
		t.Run("skips missing faces", importMultiFaces)
		t.Run("error cases", testErrorCases)
	})
}

func createCard(t *testing.T, c *card.Card) {
	t.Helper()

	withDefaults := func(c *card.Card) *card.Card {
		c.Rarity = "RARE"
		c.Border = "WHITE"
		c.Layout = "NORMAL"

		return c
	}

	cardService := card.NewService(cardDao)
	err := cardService.Import(withDefaults(c))
	require.NoError(t, err, "failed to create card")
}

func noCardMatches(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := tmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	createCard(t, &card.Card{
		CardSetCode: "20E",
		Number:      "1",
		Name:        "First",
		Faces: []*card.Face{
			{
				Name: "First",
			},
		},
	})

	importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))
	report, err := importer.Import(limitFirstPage20Entries)
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	imgCount, err := cardDao.CountImages()
	if err != nil {
		t.Fatalf("image count failed %v", err)
	}

	assert.Equal(t, 0, report.Downloaded)
	assert.Equal(t, 0, imgCount)
}

func importDifferentSets(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := tmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	createCard(t, &card.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "First",
		Faces: []*card.Face{
			{
				Name:   "First",
				Colors: card.NewColors([]string{"W", "B", "R"}),
			},
		},
	})
	createCard(t, &card.Card{
		CardSetCode: "10E",
		Number:      "2",
		Name:        "Second",
		Faces: []*card.Face{
			{
				Name:   "Second",
				Colors: card.NewColors([]string{"W"}),
			},
		},
	})
	createCard(t, &card.Card{
		CardSetCode: "9E",
		Number:      "3",
		Name:        "Third",
		Faces: []*card.Face{
			{
				Name: "Third",
			},
		},
	})

	importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))
	report, err := importer.Import(limitFirstPage20Entries)
	if err != nil {
		t.Fatalf("import failed %v", err)
	}
	imgCount, err := cardDao.CountImages()
	if err != nil {
		t.Fatalf("image count failed %v", err)
	}

	assert.Equal(t, 6, report.Downloaded)
	assert.Equal(t, 6, imgCount)
	assertFileCount(t, filepath.Join(dir, "deu", "10E"), 2)
	assertFileCount(t, filepath.Join(dir, "eng", "10E"), 2)
	assertFileCount(t, filepath.Join(dir, "eng", "9E"), 1)
	assertFileCount(t, filepath.Join(dir, "deu", "9E"), 1)
}

func hasPHashes(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := tmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
    c := &card.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "First",
		Faces: []*card.Face{
			{
				Name:   "First",
			},
		},
	}
	createCard(t, c)

	importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))
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

    assert.Greater(t, img.PHash, uint64(0))
    assert.Greater(t, img.PHashRotated, uint64(0))
}

func ignoresCardNamesCases(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))
	dir := tmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	createCard(t, &card.Card{
		CardSetCode: "10E",
		Number:      "UpPeR",
		Name:        "dIfFeReNtCaSeS",
		Faces: []*card.Face{
			{
				Name: "dIfFeReNtCaSeS",
			},
		},
	})

	importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))
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
		name    string
		fixture card.Card
		lang    string
	}{
		{
			name: "LanguageEngMissing",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "onlydeu",
				Name:        "OnlyDeu",
				Faces: []*card.Face{
					{
						Name: "OnlyDeu",
					},
				},
			},
			lang: "deu",
		},
		{
			name: "LanguageDeuMissing",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "onlyeng",
				Name:        "OnlyEng",
				Faces: []*card.Face{
					{
						Name: "OnlyEng",
					},
				},
			},
			lang: "eng",
		},
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			dir := tmpDirWithCleanup(t)
			localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir, Mode: config.CREATE})
			if err != nil {
				t.Fatalf("failed to create local storage %v", err)
			}
			createCard(t, &tc.fixture)

			importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))
			report, err := importer.Import(limitFirstPage20Entries)
			if err != nil {
				t.Fatalf("import failed %v", err)
			}
			imgCount, err := cardDao.CountImages()
			if err != nil {
				t.Fatalf("image count failed %v", err)
			}

			assert.Equal(t, 1, report.Downloaded)
			assert.Equal(t, 1, imgCount)
			assertFileCount(t, filepath.Join(dir, tc.lang, tc.fixture.CardSetCode), 1)
		})
	}
}

func importMultiFaces(t *testing.T) {
	cases := []struct {
		name    string
		fixture card.Card
		want    int
	}{
		{
			name: "FirstFaceDoesNotMatch",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "multiFace",
				Name:        "DoesNotMatch // First",
				Faces: []*card.Face{
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
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "multiFace",
				Name:        "First // DoesNotMatch",
				Faces: []*card.Face{
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
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "multiFace",
				Name:        "FirstFace // SecondFace",
				Faces: []*card.Face{
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
			dir := tmpDirWithCleanup(t)
			localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
			if err != nil {
				t.Fatalf("failed to create local storage %v", err)
			}
			createCard(t, &tc.fixture)

			importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))
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
	dir := tmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}
	createCard(t, &card.Card{
		CardSetCode: "10E",
		Number:      "1",
		Name:        "First",
		Faces: []*card.Face{
			{
				Name: "First",
			},
		},
	})
	importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))

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
	dir := tmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}

	cases := []struct {
		name    string
		fixture card.Card
	}{
		{
			name: "ExternalCardNotFound",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "99",
				Name:        "First",
				Faces: []*card.Face{
					{
						Name: "First",
					},
				},
			},
		},
		{
			name: "ExternalCardDoesNotMatch",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "1",
				Name:        "DoesNotMatch",
				Faces: []*card.Face{
					{
						Name: "DoesNotMatch",
					},
				},
			},
		},
		{
			name: "NoImageFound",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "imageNotFound",
				Name:        "ImageNotFound",
				Faces: []*card.Face{
					{
						Name: "ImageNotFound",
					},
				},
			},
		},
		{
			name: "NoImageUrlAttributeSet",
			fixture: card.Card{
				CardSetCode: "10E",
				Number:      "noImageUrl",
				Name:        "NoImageUrl",
				Faces: []*card.Face{
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
			createCard(t, &tc.fixture)

			importer := images.NewImporter(cardDao, localStorage, scryfall.NewDownloader(cfg, fetcher))

			report, err := importer.Import(limitFirstPage20Entries)
			if err != nil {
				t.Fatalf("import failed %v", err)
			}
			imgCount, err := cardDao.CountImages()
			if err != nil {
				t.Fatalf("image count failed %v", err)
			}

			assert.Equal(t, 0, report.Downloaded)
			assert.Equal(t, 0, imgCount)
			assertFileCount(t, dir, 0)
		})
	}
}

func openFile(t *testing.T, path string) (io.ReadCloser, error) {
	t.Helper()

	open, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fetch.ErrNotFound
		}
		t.Fatalf("failed to open file %s, %s", path, err)
	}

	return open, nil
}

func testdataFileFetcher(t *testing.T) *MockFetcher {
	t.Helper()

	return &MockFetcher{FakeFetch: func(url string, handleResponse func(resp *fetch.Response) error) error {
		u := strings.TrimPrefix(url, "http://localhost/")

		if strings.HasSuffix(u, ".jpg") {
			f, err := openFile(t, "testdata/"+u)
			if err != nil {
				return err
			}

			return handleResponse(&fetch.Response{
				ContentType: fetch.MimeTypeJpeg,
				Body:        f,
			})
		}

		f, err := openFile(t, "testdata/"+u)
		if err != nil {
			return err
		}

		return handleResponse(&fetch.Response{
			ContentType: fetch.MimeTypeJSON,
			Body:        f,
		})
	}}
}

func assertFileCount(t *testing.T, path string, expectedCount int) {
	t.Helper()

	files, err := os.ReadDir(path)
	if err != nil {
		t.Fatalf("failed to read dir %s %v", path, err)
	}
	assert.Equal(t, expectedCount, len(files))
}

func tmpDirWithCleanup(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "images")
	if err != nil {
		t.Fatalf("failed to create temp dir %v", err)
	}
	t.Cleanup(test.Cleanup(t, dir))

	return dir
}
