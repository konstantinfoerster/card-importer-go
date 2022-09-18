package mtgjson_test

import (
	"database/sql"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"io"
	"sort"
	"testing"
	"time"
)

var runner *postgres.DatabaseRunner

func TestImportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	runner = postgres.NewRunner()
	runner.Run(t, func(t *testing.T) {
		t.Run("CardSet: create and update", cardSetCreateAndUpdate)
		t.Run("CardSet Block: create and update", blockCreateAndUpdate)
		t.Run("CardSet Translations: create, update and remove", cardSetTranslations)
		t.Run("Card: create and update", cardCreateUpdate)
		t.Run("Card Translations: create, update and remove", cardTranslations)
		t.Run("Card all types: create, update and remove", cardTypes)
		t.Run("Card types: duplicates", duplicatedCardTypes)
	})
}

func cardSetCreateAndUpdate(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))

	csDao := cardset.NewDao(runner.Connection())
	importer := mtgjson.NewImporter(cardset.NewService(csDao), card.NewService(card.NewDao(runner.Connection())))
	want := &cardset.CardSet{
		Code:       "10E",
		Block:      cardset.CardBlock{Block: "Updated Block"},
		Name:       "Updated Name",
		TotalCount: 1,
		Released:   time.Date(2010, time.Month(8), 15, 0, 0, 0, 0, time.UTC),
		Type:       "REPRINT",
	}

	_, err := importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_update.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}

	count, _ := csDao.Count()
	assert.Equal(t, 1, count, "Unexpected set count.")
	gotSet, _ := csDao.FindCardSetByCode("10E")
	gotSet.Block.Id = sql.NullInt64{}
	assert.Equal(t, want, gotSet)
}

func blockCreateAndUpdate(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))

	csDao := cardset.NewDao(runner.Connection())
	importer := mtgjson.NewImporter(cardset.NewService(csDao), card.NewService(card.NewDao(runner.Connection())))

	_, err := importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_update.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}

	firstBlock, _ := csDao.FindBlockByName("Core Set")
	secondBlock, _ := csDao.FindBlockByName("Updated Block")
	assert.NotNil(t, firstBlock)
	assert.NotNil(t, secondBlock)
}

func cardSetTranslations(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    []*cardset.Translation
	}{
		{
			name:    "UpdateSetTranslations",
			fixture: test.LoadFile(t, "testdata/set/translations_update.json"),
			want: []*cardset.Translation{
				{
					Name: "German Translation Updated",
					Lang: "deu",
				},
			},
		},
		{
			name:    "RemoveSetTranslationsWhenNull",
			fixture: test.LoadFile(t, "testdata/set/translations_null.json"),
			want:    nil,
		},
		{
			name:    "RemoveSetTranslations",
			fixture: test.LoadFile(t, "testdata/set/translations_remove.json"),
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			csDao := cardset.NewDao(runner.Connection())
			importer := mtgjson.NewImporter(cardset.NewService(csDao), card.NewService(card.NewDao(runner.Connection())))

			_, err := importer.Import(test.LoadFile(t, "testdata/set/translations_create.json"))
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}
			_, err = importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}

			translations, _ := csDao.FindTranslations("10E")
			assert.Equal(t, tc.want, translations)
		})
	}
}

func cardCreateUpdate(t *testing.T) {
	want := []*card.Card{
		{
			CardSetCode: "2ED",
			Number:      "4",
			Name:        "Benalish Hero",
			Rarity:      "MYTHIC",
			Layout:      "TOKEN",
			Border:      "BLACK",
			Faces: []*card.Face{
				{
					Artist:            "Artist Updated",
					Colors:            card.NewColors([]string{"B", "W"}),
					ConvertedManaCost: 20,
					FlavorText:        "Flavor Text Updated",
					MultiverseId:      123,
					ManaCost:          "{B}{W}",
					Name:              "Benalish Hero",
					HandModifier:      "+2",
					LifeModifier:      "+2",
					Loyalty:           "X",
					Power:             "11",
					Toughness:         "10",
					Text:              "Text Updated",
					TypeLine:          "Type Updated",
				},
			},
		},
		{
			CardSetCode: "2ED",
			Number:      "1",
			Name:        "Second Edition Updated // The Second Updated",
			Rarity:      "RARE",
			Layout:      "SPLIT",
			Border:      "BLACK",
			Faces: []*card.Face{
				{
					ConvertedManaCost: 6,
					Name:              "Second Edition Updated",
				},
				{
					ConvertedManaCost: 8,
					Name:              "The Second Updated",
				},
			},
		},
		{
			CardSetCode: "2ED",
			Number:      "2",
			Name:        "Second Edition Face Deleted",
			Rarity:      "RARE",
			Layout:      "SPLIT",
			Border:      "WHITE",
			Faces: []*card.Face{
				{
					Name: "Second Edition Face Deleted",
				},
			},
		},
		{
			CardSetCode: "2ED",
			Number:      "3",
			Name:        "Same Face Name // Same Face Name",
			Rarity:      "RARE",
			Layout:      "REVERSIBLE_CARD",
			Border:      "BLACK",
			Faces: []*card.Face{
				{
					Name: "Same Face Name",
					Text: "Here is a text",
				},
				{
					Name: "Same Face Name",
					Text: "Here is a text",
				},
			},
		},
	}

	t.Cleanup(runner.Cleanup(t))
	cDao := card.NewDao(runner.Connection())
	imp := mtgjson.NewImporter(cardset.NewService(cardset.NewDao(runner.Connection())), card.NewService(cDao))

	_, err := imp.Import(test.LoadFile(t, "testdata/card/one_card_no_references_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = imp.Import(test.LoadFile(t, "testdata/card/one_card_no_references_update.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}

	cardCount, _ := cDao.Count()
	assert.Equal(t, 4, cardCount, "Unexpected card count.")

	for _, w := range want {
		gotCard := findUniqueCardWithReferences(t, cDao, w.CardSetCode, w.Number)
		assert.Equal(t, w, gotCard)
	}
}

func cardTranslations(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    *card.Card
	}{
		{
			name: "CreateTranslations",
			want: &card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*card.Face{
					{
						Artist: "Unknown",
						Name:   "Benalish Hero",
						Translations: []card.Translation{
							{
								Name:         "Benalische Heldin",
								Text:         "+4: Ein Spieler deiner Wahl schickt.... eine Karte aus seiner Hand ins Exil.\n−3: Schicke...",
								TypeLine:     "Legendärer Planeswalker — Karn",
								MultiverseId: 490006,
								Lang:         "deu",
							},
						},
					},
				},
			},
		},
		{
			name:    "UpdateTranslations",
			fixture: test.LoadFile(t, "testdata/card/two_translations_update.json"),
			want: &card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*card.Face{
					{
						Artist: "Unknown",
						Name:   "Benalish Hero",
						Translations: []card.Translation{
							{
								Name:         "German Name Updated",
								Text:         "German Text Updated",
								TypeLine:     "",
								MultiverseId: 123,
								Lang:         "deu",
							},
						},
					},
				},
			},
		},
		{
			name:    "RemoveTranslations",
			fixture: test.LoadFile(t, "testdata/card/two_translations_remove.json"),
			want: &card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*card.Face{
					{
						Artist: "Unknown",
						Name:   "Benalish Hero",
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))

			cDao := card.NewDao(runner.Connection())
			imp := mtgjson.NewImporter(cardset.NewService(cardset.NewDao(runner.Connection())), card.NewService(cDao))

			_, err := imp.Import(test.LoadFile(t, "testdata/card/two_translations_create.json"))
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}
			if tc.fixture != nil {
				_, err = imp.Import(tc.fixture)
				if err != nil {
					t.Fatalf("unexpected error during import %v", err)
				}
			}

			gotCard := findUniqueCardWithReferences(t, cDao, "2ED", "4")
			assert.Equal(t, tc.want, gotCard)
		})
	}
}

func cardTypes(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    *card.Card
	}{
		{
			name: "CreateTypes",
			want: &card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*card.Face{
					{
						Artist:     "Unknown",
						Name:       "Benalish Hero",
						Subtypes:   []string{"SubType1", "SubType2", "SubType3"},
						Cardtypes:  []string{"Type1", "Type2", "Type3"},
						Supertypes: []string{"SuperType1", "SuperType2", "SuperType3"},
					},
				},
			},
		},
		{
			name:    "UpdateTypes",
			fixture: test.LoadFile(t, "testdata/type/update.json"),
			want: &card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*card.Face{
					{
						Artist:     "Unknown",
						Name:       "Benalish Hero",
						Subtypes:   []string{"SubType1", "SubType3", "SubType5"},
						Cardtypes:  []string{"Type1", "Type2", "Type3", "Type4", "Type5"},
						Supertypes: []string{"SuperType1"},
					},
				},
			},
		},
		{
			name:    "RemoveTypes",
			fixture: test.LoadFile(t, "testdata/type/remove.json"),
			want: &card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*card.Face{
					{
						Artist: "Unknown",
						Name:   "Benalish Hero",
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))

			cDao := card.NewDao(runner.Connection())
			importer := mtgjson.NewImporter(cardset.NewService(cardset.NewDao(runner.Connection())), card.NewService(cDao))

			_, err := importer.Import(test.LoadFile(t, "testdata/type/create.json"))
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}
			if tc.fixture != nil {
				_, err = importer.Import(tc.fixture)
				if err != nil {
					t.Fatalf("unexpected error during import %v", err)
				}
			}

			gotCard := findUniqueCardWithReferences(t, cDao, "2ED", "4")

			assert.Equal(t, tc.want, gotCard)
		})
	}
}

func duplicatedCardTypes(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))

	cDao := card.NewDao(runner.Connection())
	importer := mtgjson.NewImporter(cardset.NewService(cardset.NewDao(runner.Connection())), card.NewService(cDao))

	_, err := importer.Import(test.LoadFile(t, "testdata/type/duplicate.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
}

func findUniqueCardWithReferences(t *testing.T, cDao *card.PostgresCardDao, setCode string, number string) *card.Card {
	c, err := cDao.FindUniqueCard(setCode, number)
	if err != nil {
		t.Fatalf("unexpected error during find unique card call %v", err)
	}

	faces, err := cDao.FindAssignedFaces(c.Id.Int64)
	if err != nil {
		t.Fatalf("unexpected error during find assigned faces call %v", err)
	}
	for _, face := range faces {
		faceId := face.Id.Int64
		translations, err := cDao.FindTranslations(faceId)
		if err != nil {
			t.Fatalf("unexpected error during find translation call %v", err)
		}
		for _, trans := range translations {
			face.Translations = append(face.Translations, *trans)
		}

		subTypes, err := cDao.FindAssignedSubTypes(faceId)
		if err != nil {
			t.Fatalf("unexpected error during find sub types call %v", err)
		}
		sort.Strings(subTypes)
		face.Subtypes = subTypes

		superTypes, err := cDao.FindAssignedSuperTypes(faceId)
		if err != nil {
			t.Fatalf("unexpected error during find super types call %v", err)
		}
		sort.Strings(superTypes)
		face.Supertypes = superTypes

		cts, err := cDao.FindAssignedCardTypes(faceId)
		if err != nil {
			t.Fatalf("unexpected error during find card types call %v", err)
		}
		sort.Strings(cts)
		face.Cardtypes = cts

		face.Id = card.PrimaryId{}
		c.Faces = append(c.Faces, face)
	}

	c.Id = card.PrimaryId{}
	return c
}
