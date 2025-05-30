package web_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	cases := []struct {
		name   string
		target string
		want   []byte
	}{
		{
			name:   "get existing file",
			target: ts.URL + "/test_file.json",
			want:   fileContent(t, filepath.Join("testdata", "test_file.json")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := web.NewClient(web.Config{}, http.DefaultClient)

			resp, err := client.Get(t.Context(), tc.target, web.NewGetOpts())
			require.NoError(t, err)
			content, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			require.NoError(t, err)
			assert.Equal(t, tc.want, content)
		})
	}
}

func TestGet_ApiError(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()
	client := web.NewClient(web.Config{}, http.DefaultClient)

	_, err := client.Get(t.Context(), ts.URL+"/notFound.unknown", web.NewGetOpts())

	var apiErr *web.ExternalAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestGet_Retry(t *testing.T) {
	cfg := web.Config{Retries: 3, Retrieables: []int{http.StatusServiceUnavailable}}
	var counter int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// 1 retry
		if counter == 2 {
			w.WriteHeader(http.StatusOK)

			return
		}

		counter++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	client := web.NewClient(cfg, http.DefaultClient)

	_, err := client.Get(t.Context(), ts.URL, web.NewGetOpts())
	assert.NoError(t, err)
	// 1 retry + first request
	assert.Equal(t, int32(2), counter)
}

func TestGet_ErrAfterMaxRetries(t *testing.T) {
	cfg := web.Config{Retries: 3, Retrieables: []int{http.StatusServiceUnavailable}}
	var counter int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		counter++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	client := web.NewClient(cfg, http.DefaultClient)

	_, err := client.Get(t.Context(), ts.URL, web.NewGetOpts())
	assert.Error(t, err)
	assert.Equal(t, int32(4), counter)
}

func TestNewGetOpts(t *testing.T) {
	want := web.GetOptions{
		Header: map[string]string{
			"content-length": "1",
		},
		StatusCodes: []int{201, 204},
	}

	actual := web.NewGetOpts().
		WithHeader("content-length", "1").
		WithExpectedCodes(201, 204)

	assert.Equal(t, want, actual)
}

func fileContent(t *testing.T, path string) []byte {
	t.Helper()

	content, err := io.ReadAll(test.LoadFile(t, path))
	require.NoError(t, err, fmt.Sprintf("failed to read data from %s", path))

	return content
}
