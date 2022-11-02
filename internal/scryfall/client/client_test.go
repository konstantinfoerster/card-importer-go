package client_test

import (
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/scryfall/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindMatchingCard(t *testing.T) {
	c := &card.Card{
		Id:   card.NewPrimaryId(1),
		Name: "First",
		Faces: []*card.Face{
			{
				Id:   card.NewPrimaryId(2),
				Name: "First",
			},
		},
	}
	sc := client.ScryfallCard{
		ImgUris: client.ScyfallImgUris{Normal: "http://localhost/first"},
		Name:    "First",
	}
	want := []*client.MatchingFace{
		{
			Url: "http://localhost/first",
			Id:  2,
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
				Id:   card.NewPrimaryId(1),
				Name: "First",
				Faces: []*card.Face{
					{
						Id:   card.NewPrimaryId(2),
						Name: "First",
					},
				},
			},
			want: []*client.MatchingFace{
				{
					Url: "http://localhost/first",
					Id:  2,
				},
			},
		},
		{
			name: "AllFacesMatch",
			fixture: card.Card{
				Id:   card.NewPrimaryId(1),
				Name: "First // Second",
				Faces: []*card.Face{
					{
						Id:   card.NewPrimaryId(2),
						Name: "First",
					},
					{
						Id:   card.NewPrimaryId(3),
						Name: "Second",
					},
				},
			},
			want: []*client.MatchingFace{
				{
					Url: "http://localhost/first",
					Id:  2,
				},
				{
					Url: "http://localhost/second",
					Id:  3,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := client.ScryfallCard{
				Name: "First // Second",
				Faces: []client.ScryfallCard{
					{
						Name:    "First",
						ImgUris: client.ScyfallImgUris{Normal: "http://localhost/first"},
					},
					{
						Name:    "Second",
						ImgUris: client.ScyfallImgUris{Normal: "http://localhost/second"},
					},
				},
			}

			parts := sc.FindMatchingCardParts(&tc.fixture)

			assert.Equal(t, tc.want, parts)
		})
	}
}
