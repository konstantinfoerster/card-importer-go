package scryfall_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/scryfall"
	"github.com/stretchr/testify/assert"
)

func TestFindURL(t *testing.T) {
	cases := []struct {
		name       string
		searchTerm string
		card       scryfall.Card
		want       string
	}{
		{
			name:       "first face match",
			searchTerm: "First",
			card: scryfall.Card{
				Name: "First // First",
				Faces: []scryfall.CardFace{
					{
						Name:    "First",
						ImgUris: scryfall.ImgURIs{Normal: "http://localhost/first"},
					},
					{
						Name:    "First",
						ImgUris: scryfall.ImgURIs{Normal: "http://localhost/second"},
					},
				},
			},
			want: "http://localhost/first",
		},
		{
			name:       "second face match",
			searchTerm: "Second",
			card: scryfall.Card{
				Name: "First // Second",
				Faces: []scryfall.CardFace{
					{
						Name:    "First",
						ImgUris: scryfall.ImgURIs{Normal: "http://localhost/first"},
					},
					{
						Name:    "Second",
						ImgUris: scryfall.ImgURIs{Normal: "http://localhost/second"},
					},
				},
			},
			want: "http://localhost/second",
		},
		{
			name:       "top card and face has url, matches first face",
			searchTerm: "First",
			card: scryfall.Card{
				Name:    "First",
				ImgUris: scryfall.ImgURIs{Normal: "http://localhost/first"},
				Faces: []scryfall.CardFace{
					{
						Name:    "First",
						ImgUris: scryfall.ImgURIs{Normal: "http://localhost/first-new"},
					},
				},
			},
			want: "http://localhost/first-new",
		},
		{
			name:       "ignore case in names",
			searchTerm: "fiRsT",
			card: scryfall.Card{
				Name:    "First",
				ImgUris: scryfall.ImgURIs{Normal: "http://localhost/first"},
				Faces: []scryfall.CardFace{
					{
						Name:    "First",
						ImgUris: scryfall.ImgURIs{Normal: "http://localhost/first-new"},
					},
				},
			},
			want: "http://localhost/first-new",
		},
		{
			name:       "fallback top card",
			searchTerm: "First",
			card: scryfall.Card{
				Name:    "different name",
				ImgUris: scryfall.ImgURIs{Normal: "http://localhost/different"},
				Faces: []scryfall.CardFace{
					{
						Name: "First",
					},
				},
			},
			want: "http://localhost/different",
		},
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			url := tc.card.FindURL(tc.searchTerm)

			assert.Equal(t, tc.want, url)
		})
	}
}
