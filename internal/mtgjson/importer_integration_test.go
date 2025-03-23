package mtgjson_test

import (
	"context"
	"database/sql"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/cards"
	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runner *postgres.DatabaseRunner

func TestImportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	runner = postgres.NewRunner()
	t.Cleanup(func() {
		ctx := context.WithoutCancel(t.Context())
		if err := runner.Stop(ctx); err != nil {
			t.Logf("failed to stop runner %v", err)
		}
	})
	err := runner.Start(t.Context())
	require.NoError(t, err)

	t.Run("CardSet: create and update", cardSetCreateAndUpdate)
	t.Run("CardSet Block: create and update", blockCreateAndUpdate)
	t.Run("CardSet Translations: create, update and remove", cardSetTranslations)
	t.Run("Card: create and update", cardCreateUpdate)
	t.Run("Card Translations: create, update and remove", cardTranslations)
	t.Run("Card all types: create, update and remove", cardTypes)
	t.Run("Card types: duplicates", duplicatedCardTypes)
}

func cardSetCreateAndUpdate(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))

	csDao := cards.NewSetDao(runner.Connection())
	importer := mtgjson.NewImporter(cards.NewSetService(csDao), cards.NewCardService(cards.NewCardDao(runner.Connection())))
	want := &cards.CardSet{
		Code:       "10E",
		Block:      cards.CardBlock{Block: "Updated Block"},
		Name:       "Updated Name",
		TotalCount: 1,
		Released:   time.Date(2010, time.Month(8), 15, 0, 0, 0, 0, time.UTC),
		Type:       "REPRINT",
	}

	_, err := importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_create.json"))
	require.NoError(t, err)

	_, err = importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_update.json"))
	require.NoError(t, err)

	count, _ := csDao.Count()
	assert.Equal(t, 1, count, "Unexpected set count.")
	gotSet, _ := csDao.FindCardSetByCode("10E")
	gotSet.Block.ID = sql.NullInt64{}
	assert.Equal(t, want, gotSet)
}

func blockCreateAndUpdate(t *testing.T) {
	t.Cleanup(runner.Cleanup(t))

	csDao := cards.NewSetDao(runner.Connection())
	importer := mtgjson.NewImporter(cards.NewSetService(csDao), cards.NewCardService(cards.NewCardDao(runner.Connection())))

	_, err := importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_create.json"))
	require.NoError(t, err)
	_, err = importer.Import(test.LoadFile(t, "testdata/set/set_no_cards_update.json"))
	require.NoError(t, err)

	firstBlock, _ := csDao.FindBlockByName("Core Set")
	secondBlock, _ := csDao.FindBlockByName("Updated Block")
	assert.NotNil(t, firstBlock)
	assert.NotNil(t, secondBlock)
}

func cardSetTranslations(t *testing.T) {
	cases := []struct {
		name   string
		source io.Reader
		want   []*cards.SetTranslation
	}{
		{
			name:   "UpdateSetTranslations",
			source: test.LoadFile(t, "testdata/set/translations_update.json"),
			want: []*cards.SetTranslation{
				{
					Name: "German Translation Updated",
					Lang: "deu",
				},
			},
		},
		{
			name:   "RemoveSetTranslationsWhenNull",
			source: test.LoadFile(t, "testdata/set/translations_null.json"),
			want:   nil,
		},
		{
			name:   "RemoveSetTranslations",
			source: test.LoadFile(t, "testdata/set/translations_remove.json"),
			want:   nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(runner.Cleanup(t))
			csDao := cards.NewSetDao(runner.Connection())
			importer := mtgjson.NewImporter(cards.NewSetService(csDao), cards.NewCardService(cards.NewCardDao(runner.Connection())))

			_, err := importer.Import(test.LoadFile(t, "testdata/set/translations_create.json"))
			require.NoError(t, err)
			_, err = importer.Import(tc.source)
			require.NoError(t, err)

			translations, _ := csDao.FindTranslations("10E")
			assert.Equal(t, tc.want, translations)
		})
	}
}

func cardCreateUpdate(t *testing.T) {
	want := []*cards.Card{
		{
			CardSetCode: "2ED",
			Number:      "4",
			Name:        "Benalish Hero",
			Rarity:      "MYTHIC",
			Layout:      "TOKEN",
			Border:      "BLACK",
			Faces: []*cards.Face{
				{
					Artist:            "Artist Updated",
					Colors:            cards.NewColors([]string{"B", "W"}),
					ConvertedManaCost: 20,
					FlavorText:        "Flavor Text Updated",
					MultiverseID:      123,
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
			Faces: []*cards.Face{
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
			Faces: []*cards.Face{
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
			Faces: []*cards.Face{
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
	cDao := cards.NewCardDao(runner.Connection())
	imp := mtgjson.NewImporter(cards.NewSetService(cards.NewSetDao(runner.Connection())), cards.NewCardService(cDao))

	_, err := imp.Import(test.LoadFile(t, "testdata/card/one_card_no_references_create.json"))
	require.NoError(t, err)
	_, err = imp.Import(test.LoadFile(t, "testdata/card/one_card_no_references_update.json"))
	require.NoError(t, err)

	cardCount, _ := cDao.Count()
	assert.Equal(t, 4, cardCount, "Unexpected card count.")

	for _, w := range want {
		gotCard := findUniqueCardWithReferences(t, cDao, w.CardSetCode, w.Number)
		assert.Equal(t, w, gotCard)
	}
}

func cardTranslations(t *testing.T) {
	cases := []struct {
		name     string
		filePath io.Reader
		want     *cards.Card
	}{
		{
			name: "CreateTranslations",
			want: &cards.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*cards.Face{
					{
						Artist: "Unknown",
						Name:   "Benalish Hero",
						Translations: []cards.FaceTranslation{
							{
								Name:         "Benalische Heldin",
								Text:         "+4: Ein Spieler deiner Wahl schickt.... eine Karte aus seiner Hand ins Exil.\n−3: Schicke...",
								TypeLine:     "Legendärer Planeswalker — Karn",
								MultiverseID: 490006,
								Lang:         "deu",
							},
						},
					},
				},
			},
		},
		{
			name:     "UpdateTranslations",
			filePath: test.LoadFile(t, "testdata/card/two_translations_update.json"),
			want: &cards.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*cards.Face{
					{
						Artist: "Unknown",
						Name:   "Benalish Hero",
						Translations: []cards.FaceTranslation{
							{
								Name:         "German Name Updated",
								Text:         "German Text Updated",
								TypeLine:     "",
								MultiverseID: 123,
								Lang:         "deu",
							},
						},
					},
				},
			},
		},
		{
			name:     "RemoveTranslations",
			filePath: test.LoadFile(t, "testdata/card/two_translations_remove.json"),
			want: &cards.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*cards.Face{
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

			cDao := cards.NewCardDao(runner.Connection())
			imp := mtgjson.NewImporter(cards.NewSetService(cards.NewSetDao(runner.Connection())), cards.NewCardService(cDao))

			_, err := imp.Import(test.LoadFile(t, "testdata/card/two_translations_create.json"))
			require.NoError(t, err)

			if tc.filePath != nil {
				_, err = imp.Import(tc.filePath)
				require.NoError(t, err)
			}

			gotCard := findUniqueCardWithReferences(t, cDao, "2ED", "4")
			assert.Equal(t, tc.want, gotCard)
		})
	}
}

func cardTypes(t *testing.T) {
	cases := []struct {
		name   string
		source io.Reader
		want   *cards.Card
	}{
		{
			name: "CreateTypes",
			want: &cards.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*cards.Face{
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
			name:   "UpdateTypes",
			source: test.LoadFile(t, "testdata/type/update.json"),
			want: &cards.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*cards.Face{
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
			name:   "RemoveTypes",
			source: test.LoadFile(t, "testdata/type/remove.json"),
			want: &cards.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []*cards.Face{
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

			cDao := cards.NewCardDao(runner.Connection())
			importer := mtgjson.NewImporter(cards.NewSetService(cards.NewSetDao(runner.Connection())), cards.NewCardService(cDao))

			_, err := importer.Import(test.LoadFile(t, "testdata/type/create.json"))
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}
			if tc.source != nil {
				_, err = importer.Import(tc.source)
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
	t.Helper()

	t.Cleanup(runner.Cleanup(t))

	cDao := cards.NewCardDao(runner.Connection())
	importer := mtgjson.NewImporter(cards.NewSetService(cards.NewSetDao(runner.Connection())), cards.NewCardService(cDao))

	_, err := importer.Import(test.LoadFile(t, "testdata/type/duplicate.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
}

func findUniqueCardWithReferences(t *testing.T, cDao *cards.PostgresCardDao, setCode string, number string) *cards.Card {
	t.Helper()

	c, err := cDao.FindUniqueCard(setCode, number)
	require.NoError(t, err, "unexpected error during find unique card call")

	faces, err := cDao.FindAssignedFaces(c.ID.Int64)
	require.NoError(t, err, "unexpected error during find assigned face call")

	for _, face := range faces {
		faceID := face.ID.Int64
		translations, err := cDao.FindTranslations(faceID)
		require.NoError(t, err, "unexpected error during find translation call")
		for _, trans := range translations {
			face.Translations = append(face.Translations, *trans)
		}

		subTypes, err := cDao.FindAssignedSubTypes(faceID)
		require.NoError(t, err, "unexpected error during find sub-types call")

		sort.Strings(subTypes)
		face.Subtypes = subTypes

		superTypes, err := cDao.FindAssignedSuperTypes(faceID)
		require.NoError(t, err, "unexpected error during find super-types call")

		sort.Strings(superTypes)
		face.Supertypes = superTypes

		cts, err := cDao.FindAssignedCardTypes(faceID)
		require.NoError(t, err, "unexpected error during find card-types call")

		sort.Strings(cts)
		face.Cardtypes = cts

		face.ID = cards.PrimaryID{}
		c.Faces = append(c.Faces, face)
	}

	c.ID = cards.PrimaryID{}

	return c
}
