package mtgjson

import (
	"context"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestParseEmptyContentFails(t *testing.T) {
	ctx := context.Background()
	r := strings.NewReader(``)
	expected := "failed to get next token"

	ch := parse(ctx, r)
	actual := <-ch

	assert.Contains(t, actual.Err.Error(), expected)
	assertChannelClosed(t, ch)
}

func TestParseInvalidJsonFails(t *testing.T) {
	ctx := context.Background()
	r := strings.NewReader(`{"data": }`)
	expected := "invalid character"

	ch := parse(ctx, r)
	actual := <-ch

	assert.Contains(t, actual.Err.Error(), expected)
	assertChannelClosed(t, ch)
}

func TestParseInvalidJsonStart(t *testing.T) {
	ctx := context.Background()
	r := strings.NewReader(`[]`)
	expected := "expected token to be"

	ch := parse(ctx, r)
	actual := <-ch

	assert.Contains(t, actual.Err.Error(), expected)
	assertChannelClosed(t, ch)
}

func TestParseSet(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		fixture io.Reader
		want    []mtgjsonCardSet
	}{
		{
			name:    "FindAllSets",
			fixture: test.LoadFile(t, "testdata/twoSetsNoCards.json"),
			want: []mtgjsonCardSet{
				{
					Code:       "10E",
					Name:       "Tenth Edition",
					Block:      "Core Set",
					Type:       "core",
					TotalCount: 383,
					Released:   "2007-07-13",
					Translations: []translation{
						{Language: "German", Name: "Hauptset Zehnte Edition"},
						{Language: "Ancient Greek", Name: ""},
						{Language: "French", Name: "10th Edition"},
					},
				},
				{
					Code: "9ED",
					Name: "Ninth Edition",
				},
			},
		},
		{
			name:    "FindNoSetsWhenRootIsEmpty",
			fixture: strings.NewReader(`{}`),
			want:    nil,
		},
		{
			name: "FindNoSetsWhenNoSetsDefined",
			fixture: strings.NewReader(`
				{
					"data": {}
				}
			`),
			want: nil,
		},
		{
			name: "FindNoSetsWhenSetHasNoData",
			fixture: strings.NewReader(`
				{
					"data": {
						"10E": {}
					}
				}
			`),
			want: nil,
		},
		{
			name: "FindSetWithEmptyTranslations",
			fixture: strings.NewReader(`
				{
					"data": {
						"10E": {
							"code": "10E",
							"translations": {}
						}
					}
				}
			`),
			want: []mtgjsonCardSet{{Code: "10E", Translations: nil}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var actual []mtgjsonCardSet
			for r := range parse(ctx, tc.fixture) {
				if r.Err != nil {
					t.Errorf("unexpected parse result, got error: %s, wanted no error", r.Err)
				}
				if tc.want == nil {
					t.Errorf("unexpected parse result, got: %v, wanted to have no result", actual)
				}

				actual = append(actual, *r.Result.(*mtgjsonCardSet))
			}

			assert.Equal(t, &tc.want, &actual)
		})
	}
}

func TestParseCards(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		fixture  io.Reader
		wantSets []string
		want     []mtgjsonCard
	}{
		{
			name:     "FindSetWithCards",
			fixture:  test.LoadFile(t, "testdata/twoSetsSetMultipleCards.json"),
			wantSets: []string{"2ED", "9ED"},
			want: []mtgjsonCard{
				{
					Artist:            "Mark Poole",
					BorderColor:       "white",
					Colors:            []string{"W"},
					ConvertedManaCost: 2.0,
					ForeignData: []foreignData{
						{
							Language:     "German",
							MultiverseId: 490006,
							Name:         "Karn der Befreite",
							Text:         "+4: Ein Spieler deiner Wahl schickt.... eine Karte aus seiner Hand ins Exil.\n−3: Schicke...",
							Type:         "Legendärer Planeswalker — Karn",
						},
						{
							Language:     "French",
							MultiverseId: 490338,
							Name:         "Karn libéré",
							Text:         "+4: Le joueur ciblé exile une carte de sa main.\n...",
							Type:         "Planeswalker légendaire : Karn",
						},
					},
					Identifiers: identifier{MultiverseId: "831"},
					Layout:      "normal",
					ManaCost:    "{1}{W}",
					Name:        "Balance",
					Number:      "3",
					Rarity:      "rare",
					Code:        "2ED",
					Subtypes:    nil,
					Supertypes:  []string{},
					Text:        "Each player chooses a number of lands they control equal...",
					Type:        "Sorcery",
					Cardtypes:   []string{"Sorcery"},
				},
				{
					Artist:            "Douglas Shuler",
					BorderColor:       "white",
					Colors:            []string{"W"},
					ConvertedManaCost: 1.0,
					FlavorText:        "Benalia has a complex caste system that changes with the...",
					ForeignData:       []foreignData{},
					Identifiers:       identifier{},
					Layout:            "normal",
					ManaCost:          "{W}",
					Name:              "Benalish Hero",
					Number:            "4",
					Power:             "1",
					Toughness:         "1",
					Rarity:            "common",
					Code:              "2ED",
					Subtypes:          []string{"Human", "Soldier"},
					Supertypes:        []string{"Test"},
					Text:              "Banding (Any creatures with banding,...",
					Type:              "Creature — Human Soldier",
					Cardtypes:         []string{"Creature"},
				},
				{
					Artist:      "Test Tester",
					Name:        "Magic Tester",
					Code:        "9ED",
					BorderColor: "black",
					Layout:      "normal",
					Number:      "1",
					Rarity:      "common",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var actualCards []mtgjsonCard
			var actualSets []string
			for r := range parse(ctx, tc.fixture) {
				if r.Err != nil {
					t.Errorf("unexpected parse result, got error: %s, wanted no error", r.Err)
				}
				if tc.want == nil {
					t.Errorf("unexpected parse result, got: %v, wanted to have no result", r.Result)
				}

				switch v := r.Result.(type) {
				case *mtgjsonCardSet:
					actualSets = append(actualSets, v.Code)
				case *mtgjsonCard:
					actualCards = append(actualCards, *v)
				default:
					t.Errorf("unknown type in parse result %v", v)
				}
			}

			if !reflect.DeepEqual(&tc.wantSets, &actualSets) {
				t.Errorf("found different set result\ngot:\t%v\nwant:\t%v", actualSets, tc.wantSets)
			}

			assert.Equal(t, &tc.want, &actualCards)
		})
	}
}

func assertChannelClosed(t *testing.T, c <-chan *result) {
	t.Helper()

	if _, ok := <-c; ok {
		t.Error("unexpected channel state. Channel is still open.")
	}
}
