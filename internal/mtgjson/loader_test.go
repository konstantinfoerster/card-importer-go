package mtgjson_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockImporter struct {
	content string
}

func (imp *mockImporter) Import(r io.Reader) (*cards.Report, error) {
	c, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	imp.content = string(c)

	return &cards.Report{}, nil
}

func TestLoad(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()
	localStorage, err := storage.NewLocalStorage(config.Storage{Location: t.TempDir()})
	require.NoErrorf(t, err, "failed to create local storate")
	cfg := config.Mtgjson{Client: web.Config{}}
	wclient := web.NewClient(cfg.Client, &http.Client{})

	cases := []struct {
		name    string
		source  string
		errPart string
		want    string
	}{
		{
			name:   "local json file",
			source: "testdata/test_file.json",
			want:   "{\"content\":\"test\"}",
		},
		{
			name:    "local file does not exists",
			source:  "testdata/doesNotExists",
			errPart: "failed to open file",
		},
		{
			name:   "external import zip",
			source: ts.URL + "/single_file.zip",
			want:   "{\"content\":\"test\"}",
		},
		{
			name:   "external import json",
			source: ts.URL + "/test_file.json",
			want:   "{\"content\":\"test\"}",
		},
		{
			name:    "external file not found",
			source:  ts.URL + "/notFound.json",
			errPart: "not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			importer := &mockImporter{}
			loader := mtgjson.NewLoader(importer, cfg, wclient, localStorage)
			u, err := url.Parse(tc.source)
			require.NoError(t, err)

			_, err = loader.Load(u)

			if tc.errPart == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errPart)
			}
			assert.Equal(t, tc.want, importer.content)
		})
	}
}
