package cards_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
)

func TestBuildFilename(t *testing.T) {
	r := cards.Image{
		MimeType: web.MimeTypeJSON,
		CardID:   cards.NewPrimaryID(1),
	}
	want := "card-1.json"

	got, err := r.BuildFilename()

	if err != nil {
		t.Fatalf("expected no error for known content type %v", err)
	}

	assert.Equal(t, want, got)
}

func TestBuildFilenameWithFaceName(t *testing.T) {
	r := cards.Image{
		MimeType: web.MimeTypeJSON,
		FaceID:   cards.NewPrimaryID(1),
	}
	want := "face-1.json"

	got, err := r.BuildFilename()

	if err != nil {
		t.Fatalf("expected no error for known content type %v", err)
	}

	assert.Equal(t, want, got)
}

func TestBuildFilenameFailsIfIdIsMissing(t *testing.T) {
	r := cards.Image{MimeType: web.MimeTypeJSON}

	_, err := r.BuildFilename()

	if err == nil {
		t.Fatal("got no error, expected an error if prefix is missing")
	}

	assert.Contains(t, err.Error(), "no valid id provided")
}

func TestBuildFilenameFailsOnUnknownContentType(t *testing.T) {
	r := cards.Image{MimeType: "unknown", CardID: cards.NewPrimaryID(1)}

	_, err := r.BuildFilename()

	if err == nil {
		t.Fatal("got no error, expected an error if content type is unknown")
	}

	assert.Contains(t, err.Error(), "unsupported mime type")
}

func TestFaceDiffWithDifferentColors(t *testing.T) {
	firstFace := cards.Face{Colors: cards.NewColors([]string{"W", "B"})}
	secFace := cards.Face{Colors: cards.NewColors([]string{"W"})}
	expected := cards.NewDiff()
	expected.Add("Colors", cards.Changes{From: firstFace.Colors, To: secFace.Colors})

	actual := firstFace.Diff(&secFace)

	assert.Equal(t, expected, actual)
}

func TestFaceDiffWithSameColors(t *testing.T) {
	firstFace := cards.Face{Colors: cards.NewColors([]string{"W"})}
	secFace := cards.Face{Colors: cards.NewColors([]string{"W"})}
	expected := cards.NewDiff()

	actual := firstFace.Diff(&secFace)

	assert.Equal(t, expected, actual)
}
