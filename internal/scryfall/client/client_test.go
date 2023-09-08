package client_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall/client"
	"github.com/stretchr/testify/assert"
)

func TestFindMatchingCard(t *testing.T) {
	c := &card.Card{
		ID:   card.NewPrimaryID(1),
		Name: "First",
		Faces: []*card.Face{
			{
				ID:   card.NewPrimaryID(2),
				Name: "First",
			},
		},
	}
	sc := client.ScryfallCard{
		ImgUris: client.ScyfallImgURIs{Normal: "http://localhost/first"},
		Name:    "First",
	}
	want := []*client.MatchingFace{
		{
			URL: "http://localhost/first",
			ID:  2,
		},
	}

	parts := sc.FindMatchingCardParts(c)

	assert.Equal(t, want, parts)
}

func TestFindMatchingFace(t *testing.T) {
	cases := []struct {
		name    string
		fixture card.Card
		want    []*client.MatchingFace
	}{
		{
			name: "FaceMatches",
			fixture: card.Card{
				ID:   card.NewPrimaryID(1),
				Name: "First",
				Faces: []*card.Face{
					{
						ID:   card.NewPrimaryID(2),
						Name: "First",
					},
				},
			},
			want: []*client.MatchingFace{
				{
					URL: "http://localhost/first",
					ID:  2,
				},
			},
		},
		{
			name: "AllFacesMatch",
			fixture: card.Card{
				ID:   card.NewPrimaryID(1),
				Name: "First // Second",
				Faces: []*card.Face{
					{
						ID:   card.NewPrimaryID(2),
						Name: "First",
					},
					{
						ID:   card.NewPrimaryID(3),
						Name: "Second",
					},
				},
			},
			want: []*client.MatchingFace{
				{
					URL: "http://localhost/first",
					ID:  2,
				},
				{
					URL: "http://localhost/second",
					ID:  3,
				},
			},
		},
	}

	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			sc := client.ScryfallCard{
				Name: "First // Second",
				Faces: []client.ScryfallCard{
					{
						Name:    "First",
						ImgUris: client.ScyfallImgURIs{Normal: "http://localhost/first"},
					},
					{
						Name:    "Second",
						ImgUris: client.ScyfallImgURIs{Normal: "http://localhost/second"},
					},
				},
			}

			parts := sc.FindMatchingCardParts(&tc.fixture)

			assert.Equal(t, tc.want, parts)
		})
	}
}
