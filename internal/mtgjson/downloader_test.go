package mtgjson

import (
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockImporter struct {
	content string
}

func (imp *mockImporter) Import(r io.Reader) (*api.Report, error) {
	c, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	imp.content = string(c)
	return &api.Report{}, nil
}

func TestDownloadableImport(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

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
			innerImporter := &mockImporter{}
			imp := NewDownloadableImport(innerImporter)

			_, err := imp.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected import error, got: %v, wanted no error", err)
			}

			assertEquals(t, tc.want, innerImporter.content)
		})
	}
}

func TestDownloadableImportFails(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

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
			name:        "ImportUnsupportedType",
			fixture:     strings.NewReader(ts.URL + "/notFound.json"),
			wantContain: "failed to download file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			innerImporter := &mockImporter{}
			imp := NewDownloadableImport(innerImporter)

			_, err := imp.Import(tc.fixture)
			if err == nil {
				t.Fatal("expected import error, but got no err")
			}

			assert.Contains(t, err.Error(), tc.wantContain)
		})
	}
}