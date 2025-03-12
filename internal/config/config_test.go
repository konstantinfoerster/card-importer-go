package config_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/stretchr/testify/assert"
)

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
