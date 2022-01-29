package mtgjson

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

var conn *postgres.DBConnection
var cleanupDB func(t *testing.T) func()

func TestImportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	runWithDatabase(t, func() {
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
	t.Cleanup(cleanupDB(t))

	csDao := cardset.NewDao(conn)
	importer := NewImporter(cardset.NewService(csDao), card.NewService(card.NewDao(conn)))
	want := &cardset.CardSet{
		Code:       "10E",
		Block:      cardset.CardBlock{Block: "Updated Block"},
		Name:       "Updated Name",
		TotalCount: 1,
		Released:   time.Date(2010, time.Month(8), 15, 0, 0, 0, 0, time.UTC),
		Type:       "REPRINT",
	}

	_, err := importer.Import(fromFile(t, "testdata/set/set_no_cards_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = importer.Import(fromFile(t, "testdata/set/set_no_cards_update.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}

	count, _ := csDao.Count()
	assert.Equal(t, 1, count, "Unexpected set count.")
	gotSet, _ := csDao.FindCardSetByCode("10E")
	gotSet.Block.Id = sql.NullInt64{}
	assertEquals(t, want, gotSet)
}

func blockCreateAndUpdate(t *testing.T) {
	t.Cleanup(cleanupDB(t))

	csDao := cardset.NewDao(conn)
	importer := NewImporter(cardset.NewService(csDao), card.NewService(card.NewDao(conn)))

	_, err := importer.Import(fromFile(t, "testdata/set/set_no_cards_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = importer.Import(fromFile(t, "testdata/set/set_no_cards_update.json"))
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
		want    []cardset.Translation
	}{
		{
			name:    "UpdateSetTranslations",
			fixture: fromFile(t, "testdata/set/translations_update.json"),
			want: []cardset.Translation{
				{
					Name: "German Translation Updated",
					Lang: "deu",
				},
			},
		},
		{
			name:    "RemoveSetTranslationsWhenNull",
			fixture: fromFile(t, "testdata/set/translations_null.json"),
			want:    nil,
		},
		{
			name:    "RemoveSetTranslations",
			fixture: fromFile(t, "testdata/set/translations_remove.json"),
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(cleanupDB(t))
			csDao := cardset.NewDao(conn)
			importer := NewImporter(cardset.NewService(csDao), card.NewService(card.NewDao(conn)))

			_, err := importer.Import(fromFile(t, "testdata/set/translations_create.json"))
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}
			_, err = importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}

			translations, _ := csDao.FindTranslations("10E")
			assertEquals(t, tc.want, translations)
		})
	}
}

func cardCreateUpdate(t *testing.T) {
	want := []card.Card{
		{
			CardSetCode: "2ED",
			Number:      "4",
			Name:        "Benalish Hero",
			Rarity:      "MYTHIC",
			Layout:      "TOKEN",
			Border:      "BLACK",
			Faces: []card.Face{
				{
					Artist:            "Artist Updated",
					Colors:            []string{"B", "W"},
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
			Faces: []card.Face{
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
			Faces: []card.Face{
				{
					Name: "Second Edition Face Deleted",
				},
			},
		},
	}

	t.Cleanup(cleanupDB(t))
	cDao := card.NewDao(conn)
	imp := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))

	_, err := imp.Import(fromFile(t, "testdata/card/one_card_no_references_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = imp.Import(fromFile(t, "testdata/card/one_card_no_references_update.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}

	cardCount, _ := cDao.Count()
	assert.Equal(t, 3, cardCount, "Unexpected card count.")

	for _, w := range want {
		gotCard := findUniqueCardWithReferences(t, cDao, w.CardSetCode, w.Number)
		assertEquals(t, w, gotCard)
	}
}

func cardTranslations(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    card.Card
	}{
		{
			name: "CreateTranslations",
			want: card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []card.Face{
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
			fixture: fromFile(t, "testdata/card/two_translations_update.json"),
			want: card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []card.Face{
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
			fixture: fromFile(t, "testdata/card/two_translations_remove.json"),
			want: card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []card.Face{
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
			t.Cleanup(cleanupDB(t))

			cDao := card.NewDao(conn)
			imp := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))

			_, err := imp.Import(fromFile(t, "testdata/card/two_translations_create.json"))
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
			assertEquals(t, tc.want, &gotCard)
		})
	}
}

func cardTypes(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    card.Card
	}{
		{
			name: "CreateTypes",
			want: card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []card.Face{
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
			fixture: fromFile(t, "testdata/type/update.json"),
			want: card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []card.Face{
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
			fixture: fromFile(t, "testdata/type/remove.json"),
			want: card.Card{
				CardSetCode: "2ED",
				Number:      "4",
				Name:        "Benalish Hero",
				Rarity:      "COMMON",
				Layout:      "NORMAL",
				Border:      "WHITE",
				Faces: []card.Face{
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
			t.Cleanup(cleanupDB(t))

			cDao := card.NewDao(conn)
			importer := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))

			_, err := importer.Import(fromFile(t, "testdata/type/create.json"))
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

			assertEquals(t, tc.want, gotCard)
		})
	}
}

func duplicatedCardTypes(t *testing.T) {
	t.Cleanup(cleanupDB(t))

	cDao := card.NewDao(conn)
	importer := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))

	_, err := importer.Import(fromFile(t, "testdata/type/duplicate.json"))
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
		for _, translation := range translations {
			face.Translations = append(face.Translations, *translation)
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

		cardTypes, err := cDao.FindAssignedCardTypes(faceId)
		if err != nil {
			t.Fatalf("unexpected error during find card types call %v", err)
		}
		sort.Strings(cardTypes)
		face.Cardtypes = cardTypes

		face.Id = sql.NullInt64{}
		c.Faces = append(c.Faces, *face)
	}

	c.Id = sql.NullInt64{}
	return c
}

func runWithDatabase(t *testing.T, runTests func()) {
	ctx := context.Background()
	err := runPostgresContainer(ctx, func(cfg *config.Database) error {
		dbConn, err := postgres.Connect(ctx, cfg)
		if err != nil {
			return err
		}
		defer func(toClose *postgres.DBConnection) {
			cErr := toClose.Close()
			if cErr != nil {
				// report close errors
				if err == nil {
					err = cErr
				} else {
					err = errors.Wrap(err, cErr.Error())
				}
			}
		}(dbConn)
		conn = dbConn

		cleanupDB = func(t *testing.T) func() {
			return func() {
				cErr := conn.Cleanup()
				if cErr != nil {
					t.Fatalf("failed to cleanup database %v", cErr)
				}
			}
		}

		runTests()

		return err
	})

	if err != nil {
		t.Fatalf("failed to start container %v", err)
	}
}

func runPostgresContainer(ctx context.Context, f func(c *config.Database) error) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get caller")
	}
	dbDirLink := filepath.Join(filepath.Dir(file), "testdata", "db")

	dbDir, err := filepath.EvalSymlinks(dbDirLink)
	if err != nil {
		return err
	}
	username := "tester"
	password := "tester"
	database := "cardmanager"

	// TODO read env variables from config
	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{"5432/tcp"},
		BindMounts: map[string]string{
			"/docker-entrypoint-initdb.d": dbDir,
		},
		Env: map[string]string{
			"POSTGRES_DB":       "postgres",
			"POSTGRES_PASSWORD": "test",
			"APP_DB_USER":       username,
			"APP_DB_PASS":       password,
			"APP_DB_NAME":       database,
		},
		WaitingFor: wait.ForLog("[1] LOG:  database system is ready to accept connections"),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}
	defer func(toClose testcontainers.Container) {
		cErr := toClose.Terminate(ctx)
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Wrap(err, cErr.Error())
			}
		}
	}(postgresC)

	if log.Debug().Enabled() {
		logs, err := postgresC.Logs(ctx)
		if err != nil {
			return err
		}
		defer logs.Close()
		b, err := ioutil.ReadAll(logs)
		if err != nil {
			return err
		}
		log.Debug().Msg(string(b))
	}

	ip, err := postgresC.Host(ctx)
	if err != nil {
		return err
	}

	mappedPort, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return err
	}

	dbConfig := &config.Database{
		Username: username,
		Password: password,
		Host:     ip,
		Port:     mappedPort.Port(),
		Database: database,
	}
	err = f(dbConfig)
	return err
}
