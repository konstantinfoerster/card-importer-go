package mtgjson_test

import (
	"io"
	"strings"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/stretchr/testify/assert"
)

func TestFileImport(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    string
	}{
		{
			name:    "ImportFromJsonFile",
			fixture: strings.NewReader("testdata/test_file.json"),
			want:    "{\"content\":\"test\"}",
		}, {
			name:    "ImportFromUnsupportedFile",
			fixture: strings.NewReader("testdata/unsupported.md"),
			want:    "#Test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeImporter := &mockImporter{}
			imp := mtgjson.NewFileDataset(fakeImporter)

			_, err := imp.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected import error, got: %v, wanted no error", err)
			}

			assert.Equal(t, tc.want, fakeImporter.content)
		})
	}
}

func TestFileImportFails(t *testing.T) {
	cases := []struct {
		name        string
		fixture     io.Reader
		wantContain string
	}{
		{
			name:        "ImportToLongFilePath",
			fixture:     strings.NewReader("SGIDJjs0neS01GDoQ2RQy0kbXXs0LFSDQuovYKjuQTN3TNEDYYyAT0RKn2oYrocsfgyPFgeQ07k2FNFc6HwIteDpeaMmUwGZ2nkQNUySoxOFcjD7FC1gVXrgBILO2TvpqOWevC2npBZD5BUigfRBflXD92QE0SwzFInWBA6BaEioC0M2qk6sPogVHCd1L5RXnaBvWV8Ye6iZIv5vLpDjKkYixF9Yl5nIHmSTE2SPe0elc66acpoDY0tVmIosQBVE"),
			wantContain: "file path must be",
		},
		{
			name:        "ImportNotExistingFile",
			fixture:     strings.NewReader("testdata/doesNotExists"),
			wantContain: "failed to open file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeImporter := &mockImporter{}
			imp := mtgjson.NewFileDataset(fakeImporter)

			_, err := imp.Import(tc.fixture)
			if err == nil {
				t.Fatal("expected import error, but got no err")
			}

			assert.Contains(t, err.Error(), tc.wantContain)
		})
	}
}
