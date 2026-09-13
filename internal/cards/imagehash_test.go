package cards_test

import (
	"image/jpeg"
	"os"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeChannelHashes(t *testing.T) {
	f, err := os.Open("testdata/images/cardImageEn.jpg")
	require.NoError(t, err)
	defer f.Close()

	img, err := jpeg.Decode(f)
	require.NoError(t, err)

	red, green, blue, err := cards.ComputeChannelPHashes(img, 16, 16)
	require.NoError(t, err)

	assert.NotEqual(t, cards.PHash{}, red)
	assert.NotEqual(t, cards.PHash{}, green)
	assert.NotEqual(t, cards.PHash{}, blue)
	assert.NotEqual(t, red, green, "red and green channel hashes should differ")
	assert.NotEqual(t, red, blue, "red and blue channel hashes should differ")
}

func TestComputeDHash(t *testing.T) {
	f, err := os.Open("testdata/images/cardImageEn.jpg")
	require.NoError(t, err)
	defer f.Close()

	img, err := jpeg.Decode(f)
	require.NoError(t, err)

	dhash, err := cards.ComputeDHash(img)
	require.NoError(t, err)

	assert.NotEqual(t, cards.DHash{}, dhash)
}
