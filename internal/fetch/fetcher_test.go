package fetch

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetch(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	allowedTypes := []string{"application/zip", "application/json"}
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
			f := NewFetcher(allowedTypes)

			result, err := f.Fetch(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected fetch error, got: %v, wanted no error", err)
			}

			assertSameFile(t, tc.want, result)
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
			f := NewFetcher([]string{})

			_, err := f.Fetch(tc.fixture)
			if err == nil {
				t.Fatal("expected import error, but got no err")
			}

			assert.Contains(t, err.Error(), tc.wantContain)
		})
	}
}

func assertSameFile(t *testing.T, expected string, f *Response) {
	defer f.Body.Close()

	got, err := io.ReadAll(f.Body)
	if err != nil {
		t.Fatalf("failed to read data, got: %v, wanted no error", err)
	}

	want, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("failed to read data, got: %v, wanted no error", err)
	}

	assert.Equal(t, want, got)
}
