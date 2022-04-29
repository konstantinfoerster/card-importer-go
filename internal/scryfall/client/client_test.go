package client

import (
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindMatchingCard(t *testing.T) {
	c := &card.Card{
		Id:   card.NewPrimaryId(1),
		Name: "First",
		Faces: []*card.Face{
			{
				Name: "First",
			},
		},
	}
	sc := ScryfallCard{
		ImgUris: ScyfallImgUris{Normal: "http://localhost/first"},
		Name:    "First",
	}
	want := []*MatchedPart{
		{
			Url:         "http://localhost/first",
			CardId:      1,
			MatchedType: "CARD",
			MatchedId:   1,
		},
	}

	parts := sc.FindMatchingCardParts(c)

	assert.Equal(t, want, parts)
}

func TestFindMatchingFace(t *testing.T) {
	cases := []struct {
		name    string
		fixture card.Card
		want    []*MatchedPart
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
			want: []*MatchedPart{
				{
					Url:         "http://localhost/first",
					CardId:      1,
					MatchedType: "FACE",
					MatchedId:   2,
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
			want: []*MatchedPart{
				{
					Url:         "http://localhost/first",
					CardId:      1,
					MatchedType: "FACE",
					MatchedId:   2,
				},
				{
					Url:         "http://localhost/second",
					CardId:      1,
					MatchedType: "FACE",
					MatchedId:   3,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := ScryfallCard{
				Name: "First // Second",
				Faces: []ScryfallCard{
					{
						Name:    "First",
						ImgUris: ScyfallImgUris{Normal: "http://localhost/first"},
					},
					{
						Name:    "Second",
						ImgUris: ScyfallImgUris{Normal: "http://localhost/second"},
					},
				},
			}

			parts := sc.FindMatchingCardParts(&tc.fixture)

			assert.Equal(t, tc.want, parts)
		})
	}
}
