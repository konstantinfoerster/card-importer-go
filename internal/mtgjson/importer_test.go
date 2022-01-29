package mtgjson

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/stretchr/testify/assert"
	"io"
	"sort"
	"testing"
)

type MockSetService struct {
	Sets       []cardset.CardSet
	FakeImport func(count int, set *cardset.CardSet) error
}

func (s *MockSetService) Import(set *cardset.CardSet) error {
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
	Cards      []card.Card
	FakeImport func(count int, set *card.Card) error
}

func (s *MockCardService) Import(card *card.Card) error {
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

func (s *MockCardService) CardsOrdered() []card.Card {
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
		FakeImport: func(count int, c *card.Card) error {
			if count > 0 {
				return fmt.Errorf("card import failed [%s]", c.Number)
			}
			return nil
		},
	}
	wantCards := 1

	importer := NewImporter(&setService, &cardService)
	_, err := importer.Import(fromFile(t, "testdata/twoSetsSetMultipleCards.json"))

	assert.Contains(t, err.Error(), "card import failed")

	if len(cardService.Cards) != wantCards {
		t.Errorf("unexpected card count, got: %d, wanted: %d", len(cardService.Cards), wantCards)
	}
}

func TestImportSetsWithImportError(t *testing.T) {
	setService := MockSetService{
		FakeImport: func(count int, c *cardset.CardSet) error {
			if count > 0 {
				return fmt.Errorf("set import failed [%s]", c.Code)
			}
			return nil
		},
	}
	cardService := MockCardService{}
	wantSets := 1

	importer := NewImporter(&setService, &cardService)
	_, err := importer.Import(fromFile(t, "testdata/twoSetsSetMultipleCards.json"))

	assert.Contains(t, err.Error(), "set import failed")
	if len(setService.Sets) != wantSets {
		t.Errorf("unexpected set count, got: %d, wanted: %d", len(cardService.Cards), wantSets)
	}
}

func TestImportCards(t *testing.T) {
	cases := []struct {
		name     string
		fixture  io.Reader
		wantSets int
		want     []card.Card
	}{
		{
			name:     "ImportMultipleCards",
			fixture:  fromFile(t, "testdata/twoSetsSetMultipleCards.json"),
			wantSets: 2,
			want: []card.Card{
				{
					CardSetCode: "9ED",
					Number:      "1",
					Name:        "Magic Tester",
					Rarity:      "COMMON",
					Layout:      "NORMAL",
					Border:      "BLACK",
					Faces: []card.Face{
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
					Faces: []card.Face{
						{
							Name:              "Balance",
							Artist:            "Mark Poole",
							ConvertedManaCost: 2.0,
							Colors:            []string{"W"},
							Text:              "Each player chooses a number of lands they control equal...",
							ManaCost:          "{1}{W}",
							TypeLine:          "Sorcery",
							Cardtypes:         []string{"Sorcery"},
							MultiverseId:      831,
							Translations: []card.Translation{
								{
									Name:         "Karn der Befreite",
									Text:         "+4: Ein Spieler deiner Wahl schickt.... eine Karte aus seiner Hand ins Exil.\n−3: Schicke...",
									TypeLine:     "Legendärer Planeswalker — Karn",
									MultiverseId: 490006,
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
					Faces: []card.Face{
						{
							Name:              "Benalish Hero",
							Artist:            "Douglas Shuler",
							ConvertedManaCost: 1.0,
							Colors:            []string{"W"},
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
			importer := NewImporter(&setService, &cardService)

			_, err := importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected import error, got: %v, wanted no error", err)
			}
			if len(setService.Sets) != tc.wantSets {
				t.Fatalf("unexpected set count, got: %d, wanted %d", len(setService.Sets), tc.wantSets)
			}
			if len(cardService.Cards) != len(tc.want) {
				t.Fatalf("unexpected card count, got: %d, wanted %d", len(cardService.Cards), len(tc.want))
			}

			assertEquals(t, tc.want, cardService.CardsOrdered())
		})
	}
}

func TestImportCardWithMultipleFaces(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    []card.Card
	}{
		{
			name:    "ImportMultipleCardsWithMultipleFaces",
			fixture: fromFile(t, "testdata/card/multiple_cards_multiple_faces.json"),
			want: []card.Card{
				{
					CardSetCode: "2ED",
					Number:      "3",
					Name:        "One // Two // Three // Four // Five // Six // Sven",
					Rarity:      "RARE",
					Layout:      "NORMAL",
					Border:      "WHITE",
					Faces: []card.Face{
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
					Faces: []card.Face{
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
					Faces: []card.Face{
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
			importer := NewImporter(&setService, &cardService)

			_, err := importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected import error, got: %v, wanted no error", err)
			}
			if len(cardService.Cards) != len(tc.want) {
				t.Fatalf("unexpected card count, got: %d, wanted %d", len(cardService.Cards), len(tc.want))
			}

			assertEquals(t, tc.want, cardService.CardsOrdered())
		})
	}
}

func TestImportCardWithInvalidFaces(t *testing.T) {
	cases := []struct {
		name        string
		fixture     io.Reader
		wantContain string
	}{
		{
			name:        "ImportCardsWithInvalidFaces",
			fixture:     fromFile(t, "testdata/card/cards_invalid_faces.json"),
			wantContain: "unprocessed double face cards",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setService := MockSetService{}
			cardService := MockCardService{}
			importer := NewImporter(&setService, &cardService)

			_, err := importer.Import(tc.fixture)
			if err == nil {
				t.Fatalf("expected import error to contain %v, got no error", tc.wantContain)
			}
			if len(cardService.Cards) != 0 {
				t.Fatalf("unexpected card count, got: %d, wanted 0", len(cardService.Cards))
			}

			assert.Contains(t, err.Error(), tc.wantContain)
		})
	}
}
