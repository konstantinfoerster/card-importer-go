package fetch

import (
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var cfg = config.Http{Timeout: 5 * time.Second}

func TestFetch(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	cases := []struct {
		name    string
		fixture string
		want    string
	}{
		{
			name:    "FetchZip",
			fixture: ts.URL + "/test_file.zip",
			want:    filepath.Join("testdata", "test_file.zip"),
		},
		{
			name:    "FetchJson",
			fixture: ts.URL + "/test_file.json",
			want:    filepath.Join("testdata", "test_file.json"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFetcher(cfg)

			var got []byte
			var err error
			err = f.Fetch(tc.fixture, func(r *Response) error {
				got, err = io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read data, got: %v, wanted no error", err)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected fetch error, got: %v, wanted no error", err)
			}

			assertSameFile(t, tc.want, got)
		})
	}
}

func TestFetchFails(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	cases := []struct {
		name        string
		fixture     string
		wantContain string
	}{
		{
			name:        "FetchUnsupportedType",
			fixture:     ts.URL + "/test_file.json",
			wantContain: "unsupported content-type",
		},
		{
			name:        "FetchNoneExistingFile",
			fixture:     ts.URL + "/notFound.unknown",
			wantContain: "404",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFetcher(cfg, NewContentTypeValidator([]string{}))

			err := f.Fetch(tc.fixture, func(resp *Response) error {
				return nil
			})
			if err == nil {
				t.Fatal("expected import error, but got no err")
			}

			assert.Contains(t, err.Error(), tc.wantContain)
		})
	}
}

func assertSameFile(t *testing.T, want string, got []byte) {
	wantContent, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("failed to read data, got: %v, wanted no error", err)
	}

	assert.Equal(t, wantContent, got)
}
