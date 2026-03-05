package config_test

import (
	"testing"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfigs_EmptyFile(t *testing.T) {
	_, err := config.ReadConfigs("testdata/empty.yaml")

	require.NoError(t, err)
}

func TestReadConfigs_MultipleFilesOverwrite(t *testing.T) {
	cfg, err := config.ReadConfigs(
		"testdata/application.yaml",
		"testdata/application-dev.yaml",
	)

	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, "tester", cfg.Database.Username)
	assert.Equal(t, "s3cr3t", cfg.Database.Password)
	assert.Equal(t, "https://localhost:8443/cards.zip", cfg.Mtgjson.DatasetURL)
	assert.Equal(t, time.Second*120, cfg.Mtgjson.Client.Timeout)
}

func TestDatabase_ConnectionURL(t *testing.T) {
	cfg := config.Database{
		Host:     "localhost",
		Port:     "5432",
		Database: "test",
		Username: "Tester",
		Password: "secret",
	}

	assert.Equal(t, "postgres://Tester:secret@localhost:5432/test", cfg.ConnectionURL())
}

func TestReadConfigs_NotAFile(t *testing.T) {
	cases := []struct {
		name string
		path []string
	}{
		{
			name: "directory",
			path: []string{"testdata"},
		},
		{
			name: "file not exist",
			path: []string{"testdata/notfound.yaml"},
		},
		{
			name: "second file not exist",
			path: []string{
				"testdata/application.yaml",
				"testdata/notfound.yaml",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.ReadConfigs(tc.path...)

			require.ErrorIs(t, err, config.ErrReadFile)
		})
	}
}

func TestEnsureBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr error
		want    string
	}{
		{
			name: "url with http schema",
			url:  "http://localhost/a/b",
			want: "http://localhost/a/b",
		},
		{
			name: "url with schema and query",
			url:  "http://localhost/a/b?param=true",
			want: "http://localhost/a/b?param=true",
		},
		{
			name: "url with ftp schema",
			url:  "ftp://localhost/a/b",
			want: "ftp://localhost/a/b",
		},
		{
			name: "relative url",
			url:  "a/b",
			want: "http://localhost/a/b",
		},
		{
			name: "relative url with query",
			url:  "a/b?param=true",
			want: "http://localhost/a/b?param=true",
		},
		{
			name: "absolute url without schema",
			url:  "/a/b",
			want: "http://localhost/a/b",
		},
		{
			name: "absolute url without schema with query",
			url:  "/a/b?param=true",
			want: "http://localhost/a/b?param=true",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Scryfall{BaseURL: "http://localhost"}

			actual, err := cfg.EnsureBaseURL(tc.url)

			assert.Equal(t, tc.want, actual)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
