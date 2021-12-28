package mtgjson

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/stretchr/testify/assert"
	"io"
	"sort"
	"testing"
	"time"
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

func TestImportCardsWithImportError(t *testing.T) {
	setService := MockSetService{}
	cardService := MockCardService{
		FakeImport: func(count int, c *card.Card) error {
			if count > 0 {
				return fmt.Errorf("card import failed [%s]", c.Name)
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

func TestImportSets(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    []cardset.CardSet
	}{
		{
			name:    "ImportMultipleSets",
			fixture: fromFile(t, "testdata/twoSetsNoCards.json"),
			want: []cardset.CardSet{
				{
					Code:       "10E",
					Name:       "Tenth Edition",
					TotalCount: 383,
					Released:   time.Date(2007, time.Month(7), 13, 0, 0, 0, 0, time.UTC),
					Block:      cardset.CardBlock{Block: "Core Set"},
					Type:       "CORE",
					Translations: []cardset.Translation{
						{Name: "Hauptset Zehnte Edition", Lang: "deu"},
					},
				},
				{
					Code: "9ED",
					Name: "Ninth Edition",
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
			assertEquals(t, tc.want, setService.Sets)
		})
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
					Name:              "Balance",
					Artist:            "Mark Poole",
					Border:            "WHITE",
					ConvertedManaCost: 2.0,
					Colors:            []string{"W"},
					Text:              "Each player chooses a number of lands they control equal...",
					Layout:            "NORMAL",
					ManaCost:          "{1}{W}",
					Rarity:            "RARE",
					Number:            "3",
					FullType:          "Sorcery",
					Cardtypes:         []string{"Sorcery"},
					CardSetCode:       "2ED",
					MultiverseId:      831,
					Translations: []card.Translation{
						{
							Name:         "Karn der Befreite",
							Text:         "+4: Ein Spieler deiner Wahl schickt.... eine Karte aus seiner Hand ins Exil.\n−3: Schicke...",
							FullType:     "Legendärer Planeswalker — Karn",
							MultiverseId: 490006,
							Lang:         "deu",
						},
					},
				},
				{
					Name:              "Benalish Hero",
					Artist:            "Douglas Shuler",
					Border:            "WHITE",
					ConvertedManaCost: 1.0,
					Colors:            []string{"W"},
					Text:              "Banding (Any creatures with banding,...",
					FlavorText:        "Benalia has a complex caste system that changes with the...",
					Layout:            "NORMAL",
					ManaCost:          "{W}",
					Number:            "4",
					Power:             "1",
					Toughness:         "1",
					Rarity:            "COMMON",
					Subtypes:          []string{"Human", "Soldier"},
					Supertypes:        []string{"Test"},
					Cardtypes:         []string{"Creature"},
					FullType:          "Creature — Human Soldier",
					CardSetCode:       "2ED",
				},
				{
					Name:        "Magic Tester",
					Artist:      "Test Tester",
					CardSetCode: "9ED",
					Border:      "BLACK",
					Layout:      "NORMAL",
					Number:      "1",
					Rarity:      "COMMON",
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

			// bring into same order
			sort.Slice(tc.want, func(i, j int) bool {
				return tc.want[i].Name < tc.want[j].Name
			})
			sort.Slice(cardService.Cards, func(i, j int) bool {
				return cardService.Cards[i].Name < cardService.Cards[j].Name
			})
			assertEquals(t, tc.want, cardService.Cards)
		})
	}
}
