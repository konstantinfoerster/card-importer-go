package mtgjson_test

import (
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var cfg = config.Http{Timeout: 5 * time.Second}

type mockImporter struct {
	content string
}

func (imp *mockImporter) Import(r io.Reader) (*dataset.Report, error) {
	c, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	imp.content = string(c)
	return &dataset.Report{}, nil
}

func TestDownloadableImport(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()
	dir := test.NewTmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}

	cases := []struct {
		name    string
		fixture io.Reader
		want    string
	}{
		{
			name:    "ImportFromZip",
			fixture: strings.NewReader(ts.URL + "/single_file.zip"),
			want:    "{\"content\":\"test\"}",
		},
		{
			name:    "ImportFromJson",
			fixture: strings.NewReader(ts.URL + "/test_file.json"),
			want:    "{\"content\":\"test\"}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeImporter := &mockImporter{}
			imp := mtgjson.NewDownloadableDataset(fakeImporter, fetch.NewFetcher(cfg), localStorage)

			_, err := imp.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected import error, got: %v, wanted no error", err)
			}

			assert.Equal(t, tc.want, fakeImporter.content)
		})
	}
}

func TestDownloadableImportFails(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()
	dir := test.NewTmpDirWithCleanup(t)
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: dir})
	if err != nil {
		t.Fatalf("failed to create local storage %v", err)
	}

	cases := []struct {
		name        string
		fixture     io.Reader
		wantContain string
	}{
		{
			name:        "ImportFromZipWithMultiFiles",
			fixture:     strings.NewReader(ts.URL + "/two_files.zip"),
			wantContain: "unexpected file count",
		},
		{
			name:        "ImportUnsupportedType",
			fixture:     strings.NewReader(ts.URL + "/unsupported.md"),
			wantContain: "unsupported content-type",
		},
		{
			name:        "ImportNoneExistingFile",
			fixture:     strings.NewReader(ts.URL + "/notFound.json"),
			wantContain: "not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeImporter := &mockImporter{}
			validator := fetch.NewContentTypeValidator([]string{fetch.MimeTypeZip, fetch.MimeTypeJson})
			imp := mtgjson.NewDownloadableDataset(fakeImporter, fetch.NewFetcher(cfg, validator), localStorage)

			_, err := imp.Import(tc.fixture)
			if err == nil {
				t.Fatal("expected import error, but got no err")
			}

			assert.Contains(t, err.Error(), tc.wantContain)
		})
	}
}
