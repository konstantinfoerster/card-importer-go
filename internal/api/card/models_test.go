package card_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/diff"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/stretchr/testify/assert"
)

func TestBuildFilename(t *testing.T) {
	r := card.Image{
		MimeType: fetch.MimeTypeJSON,
		CardID:   card.NewPrimaryID(1),
	}
	want := "card-1.json"

	got, err := r.BuildFilename()

	if err != nil {
		t.Fatalf("expected no error for known content type %v", err)
	}

	assert.Equal(t, want, got)
}

func TestBuildFilenameWithFaceName(t *testing.T) {
	r := card.Image{
		MimeType: fetch.MimeTypeJSON,
		FaceID:   card.NewPrimaryID(1),
	}
	want := "face-1.json"

	got, err := r.BuildFilename()

	if err != nil {
		t.Fatalf("expected no error for known content type %v", err)
	}

	assert.Equal(t, want, got)
}

func TestBuildFilenameFailsIfIdIsMissing(t *testing.T) {
	r := card.Image{MimeType: fetch.MimeTypeJSON}

	_, err := r.BuildFilename()

	if err == nil {
		t.Fatal("got no error, expected an error if prefix is missing")
	}

	assert.Contains(t, err.Error(), "no valid id provided")
}

func TestBuildFilenameFailsOnUnknownContentType(t *testing.T) {
	r := card.Image{MimeType: "unknown", CardID: card.NewPrimaryID(1)}

	_, err := r.BuildFilename()

	if err == nil {
		t.Fatal("got no error, expected an error if content type is unknown")
	}

	assert.Contains(t, err.Error(), "unsupported mime type")
}

func TestFaceDiffWithDifferentColors(t *testing.T) {
	firstFace := card.Face{Colors: card.NewColors([]string{"W", "B"})}
	secFace := card.Face{Colors: card.NewColors([]string{"W"})}
	expected := diff.New()
	expected.Add("Colors", diff.Changes{From: firstFace.Colors, To: secFace.Colors})

	actual := firstFace.Diff(&secFace)

	assert.Equal(t, expected, actual)
}

func TestFaceDiffWithSameColors(t *testing.T) {
	firstFace := card.Face{Colors: card.NewColors([]string{"W"})}
	secFace := card.Face{Colors: card.NewColors([]string{"W"})}
	expected := diff.New()

	actual := firstFace.Diff(&secFace)

	assert.Equal(t, expected, actual)
}
