package mtgjson_test

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO Revisit this tests

type MockSetService struct {
	Sets       []cards.CardSet
	FakeImport func(count int, set *cards.CardSet) error
}

func (s *MockSetService) Import(set *cards.CardSet) error {
	if s.FakeImport != nil {
		if err := s.FakeImport(len(s.Sets), set); err != nil {
			return err
		}
	}
	s.Sets = append(s.Sets, *set)

	return nil
}
func (s *MockSetService) Count() (int, error) {
	return len(s.Sets), nil
}

type MockCardService struct {
	mu         sync.Mutex
	Cards      []cards.Card
	FakeImport func(count int, set *cards.Card) error
}

func (s *MockCardService) Import(card *cards.Card) error {
	// will be called concurrently from the importer
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.FakeImport != nil {
		if err := s.FakeImport(len(s.Cards), card); err != nil {
			return err
		}
	}

	s.Cards = append(s.Cards, *card)

	return nil
}

func (s *MockCardService) Count() (int, error) {
	return len(s.Cards), nil
}

func (s *MockCardService) CardsOrdered() []cards.Card {
	// bring cards into same order
	sort.SliceStable(s.Cards, func(i, j int) bool {
		return s.Cards[i].Number < s.Cards[j].Number
	})
	// bring card faces into same order
	for _, c := range s.Cards {
		sort.SliceStable(c.Faces, func(i, j int) bool {
			return c.Faces[i].Name < c.Faces[j].Name
		})
	}

	return s.Cards
}

func TestImportCardsWithImportError(t *testing.T) {
	setService := MockSetService{}
	cardService := MockCardService{
		FakeImport: func(count int, c *cards.Card) error {
			if count > 0 {
				return fmt.Errorf("card import failed [%s]", c.Number)
			}

			return nil
		},
	}
	wantCount := 1

	importer := mtgjson.NewImporter(&setService, &cardService)
	_, err := importer.Import(test.LoadFile(t, "testdata/twoSetsSetMultipleCards.json"))

	assert.ErrorContains(t, err, "card import failed")
	assert.Len(t, cardService.Cards, wantCount, "unexpected card count")
}

func TestImportSetsWithImportError(t *testing.T) {
	setService := MockSetService{
		FakeImport: func(count int, c *cards.CardSet) error {
			if count > 0 {
				return fmt.Errorf("set import failed [%s]", c.Code)
			}

			return nil
		},
	}
	cardService := MockCardService{}
	wantCount := 1

	importer := mtgjson.NewImporter(&setService, &cardService)
	_, err := importer.Import(test.LoadFile(t, "testdata/twoSetsSetMultipleCards.json"))

	assert.ErrorContains(t, err, "set import failed")
	assert.Len(t, setService.Sets, wantCount, "unexpected set count")
}

func TestImportCards(t *testing.T) {
	cases := []struct {
		name     string
		source   io.Reader
		wantSets int
		want     []cards.Card
	}{
		{
			name:     "ImportMultipleCards",
			source:   test.LoadFile(t, "testdata/twoSetsSetMultipleCards.json"),
			wantSets: 2,
			want: []cards.Card{
				{
					CardSetCode: "9ED",
					Number:      "1",
					Name:        "Magic Tester",
					Rarity:      "COMMON",
					Layout:      "NORMAL",
					Border:      "BLACK",
					Faces: []*cards.Face{
						{
							Name:   "Magic Tester",
							Artist: "Test Tester",
						},
					},
				},
				{
					CardSetCode: "2ED",
					Number:      "3",
					Name:        "Balance",
					Rarity:      "RARE",
					Layout:      "NORMAL",
					Border:      "WHITE",
					Faces: []*cards.Face{
						{
							Name:              "Balance",
							Artist:            "Mark Poole",
							ConvertedManaCost: 2.0,
							Colors:            cards.NewColors([]string{"W"}),
							Text:              "Each player chooses a number of lands they control equal...",
							ManaCost:          "{1}{W}",
							TypeLine:          "Sorcery",
							Cardtypes:         []string{"Sorcery"},
							MultiverseID:      831,
							Translations: []cards.FaceTranslation{
								{
									Name:         "Karn der Befreite",
									Text:         "+4: Ein Spieler deiner Wahl schickt.... eine Karte aus seiner Hand ins Exil.\n−3: Schicke...",
									TypeLine:     "Legendärer Planeswalker — Karn",
									MultiverseID: 490006,
									Lang:         "deu",
								},
							},
						},
					},
				},
				{
					CardSetCode: "2ED",
					Number:      "4",
					Rarity:      "COMMON",
					Name:        "Benalish Hero",
					Layout:      "NORMAL",
					Border:      "WHITE",
					Faces: []*cards.Face{
						{
							Name:              "Benalish Hero",
							Artist:            "Douglas Shuler",
							ConvertedManaCost: 1.0,
							Colors:            cards.NewColors([]string{"W"}),
							Text:              "Banding (Any creatures with banding,...",
							FlavorText:        "Benalia has a complex caste system that changes with the...",
							ManaCost:          "{W}",
							Power:             "1",
							Toughness:         "1",
							Subtypes:          []string{"Human", "Soldier"},
							Supertypes:        []string{"Test"},
							TypeLine:          "Creature — Human Soldier",
							Cardtypes:         []string{"Creature"},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setService := MockSetService{}
			cardService := MockCardService{}
			importer := mtgjson.NewImporter(&setService, &cardService)

			_, err := importer.Import(tc.source)

			require.NoError(t, err)
			require.Len(t, setService.Sets, tc.wantSets, "unexpected set count")
			require.Len(t, cardService.Cards, len(tc.want), "unexpected card count")
			assert.Equal(t, tc.want, cardService.CardsOrdered())
		})
	}
}

func TestImportCardWithMultipleFaces(t *testing.T) {
	cases := []struct {
		name   string
		source io.Reader
		want   []cards.Card
	}{
		{
			name:   "ImportMultipleCardsWithMultipleFaces",
			source: test.LoadFile(t, "testdata/card/multiple_cards_multiple_faces.json"),
			want: []cards.Card{
				{
					CardSetCode: "2ED",
					Number:      "3",
					Name:        "One // Two // Three // Four // Five // Six // Sven",
					Rarity:      "RARE",
					Layout:      "NORMAL",
					Border:      "WHITE",
					Faces: []*cards.Face{
						{
							Name:              "Five",
							ConvertedManaCost: 5.0,
						}, {
							Name:              "Four",
							ConvertedManaCost: 4.0,
						},
						{
							Name:              "One",
							ConvertedManaCost: 1.0,
						}, {
							Name:              "Seven",
							ConvertedManaCost: 7.0,
						}, {
							Name:              "Six",
							ConvertedManaCost: 6.0,
						}, {
							Name:              "Three",
							ConvertedManaCost: 3.0,
						}, {
							Name:              "Two",
							ConvertedManaCost: 2.0,
						},
					},
				},
				{
					CardSetCode: "2ED",
					Number:      "4",
					Rarity:      "RARE",
					Name:        "1 / 2",
					Layout:      "NORMAL",
					Border:      "WHITE",
					Faces: []*cards.Face{
						{
							Name: "1 / 2",
						},
					},
				},
				{
					CardSetCode: "2ED",
					Number:      "5",
					Rarity:      "RARE",
					Name:        "First // Second",
					Layout:      "MELD",
					Border:      "WHITE",
					Faces: []*cards.Face{
						{
							Name: "First",
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setService := MockSetService{}
			cardService := MockCardService{}
			importer := mtgjson.NewImporter(&setService, &cardService)

			_, err := importer.Import(tc.source)

			require.NoError(t, err)
			require.Len(t, cardService.Cards, len(tc.want), "unexpected card count")
			assert.Equal(t, tc.want, cardService.CardsOrdered())
		})
	}
}

func TestImportCardWithInvalidFaces(t *testing.T) {
	cases := []struct {
		name        string
		source      io.Reader
		wantContain string
	}{
		{
			name:        "ImportCardsWithInvalidFaces",
			source:      test.LoadFile(t, "testdata/card/cards_invalid_faces.json"),
			wantContain: "unprocessed double face cards",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setService := MockSetService{}
			cardService := MockCardService{}
			importer := mtgjson.NewImporter(&setService, &cardService)

			_, err := importer.Import(tc.source)

			require.Error(t, err)
			require.Zero(t, len(cardService.Cards), "unexpected card count")
			assert.ErrorContains(t, err, tc.wantContain)
		})
	}
}
