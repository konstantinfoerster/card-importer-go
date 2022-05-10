package card

import (
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildFilename(t *testing.T) {
	r := CardImage{
		MimeType: fetch.NewMimeType(fetch.MimeTypeJson),
		CardId:   NewPrimaryId(1),
	}
	want := "card-1.json"

	got, err := r.BuildFilename()

	if err != nil {
		t.Fatalf("expected no error for known content type %v", err)
	}

	assert.Equal(t, want, got)
}

func TestBuildFilenameWithFaceName(t *testing.T) {
	r := CardImage{
		MimeType: fetch.NewMimeType(fetch.MimeTypeJson),
		FaceId:   NewPrimaryId(1),
	}
	want := "face-1.json"

	got, err := r.BuildFilename()

	if err != nil {
		t.Fatalf("expected no error for known content type %v", err)
	}

	assert.Equal(t, want, got)
}

func TestBuildFilenameFailsIfIdIsMissing(t *testing.T) {
	r := CardImage{MimeType: fetch.NewMimeType(fetch.MimeTypeJson)}

	_, err := r.BuildFilename()

	if err == nil {
		t.Fatal("got no error, expected an error if prefix is missing")
	}

	assert.Contains(t, err.Error(), "no valid id provided")
}

func TestBuildFilenameFailsOnUnknownContentType(t *testing.T) {
	r := CardImage{MimeType: fetch.NewMimeType("unknown"), CardId: NewPrimaryId(1)}

	_, err := r.BuildFilename()

	if err == nil {
		t.Fatal("got no error, expected an error if content type is unknown")
	}

	assert.Contains(t, err.Error(), "unsupported mime type")
}
