package mtgjson

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
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
		t.Run("Card: duplicates", duplicatedCards)
		t.Run("Card Translations: create, update and remove", cardTranslations)
		t.Run("Card Card-Type: create, update and remove", cardCardTypes)
		t.Run("Card Super-Type: create, update and remove", cardSuperTypes)
		t.Run("Card Sub-Type: create, update and remove", cardSubTypes)
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

			_, err := importer.Import(fromFile(t, "testdata/card/two_translations_create.json"))
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
	t.Cleanup(cleanupDB(t))

	cDao := card.NewDao(conn)
	importer := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))
	want := &card.Card{
		Artist:            "Artist Updated",
		Border:            "BLACK",
		Colors:            []string{"B", "W"},
		ConvertedManaCost: 20,
		FlavorText:        "Flavor Text Updated",
		MultiverseId:      123,
		Layout:            "FLIP",
		ManaCost:          "{B}{W}",
		Name:              "Benalish Hero",
		HandModifier:      2,
		LifeModifier:      2,
		Loyalty:           "X",
		Number:            "4",
		Power:             "10",
		Toughness:         "10",
		Rarity:            "MYTHIC",
		CardSetCode:       "2ED",
		Text:              "Text Updated",
		FullType:          "Type Updated",
	}

	_, err := importer.Import(fromFile(t, "testdata/card/one_card_no_references_create.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
	_, err = importer.Import(fromFile(t, "testdata/card/one_card_no_references_update.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}

	cardCount, _ := cDao.Count()
	assert.Equal(t, 1, cardCount, "Unexpected card count.")
	gotCard, _ := cDao.FindUniqueCard("Benalish Hero", "2ED", "4")
	gotCard.Id = sql.NullInt64{}
	assertEquals(t, want, gotCard)
}

func duplicatedCards(t *testing.T) {
	t.Cleanup(cleanupDB(t))

	cDao := card.NewDao(conn)
	importer := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))

	_, err := importer.Import(fromFile(t, "testdata/card/duplicate_card.json"))
	if err != nil {
		t.Fatalf("unexpected error during import %v", err)
	}
}

func cardTranslations(t *testing.T) {
	cases := []struct {
		name    string
		fixture io.Reader
		want    []card.Translation
	}{
		{
			name:    "UpdateTranslations",
			fixture: fromFile(t, "testdata/card/two_translations_update.json"),
			want: []card.Translation{
				{
					Name:         "German Name Updated",
					Text:         "German Text Updated",
					FullType:     "",
					MultiverseId: 123,
					Lang:         "deu",
				},
			},
		},
		{
			name:    "RemoveTranslations",
			fixture: fromFile(t, "testdata/card/two_translations_remove.json"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(cleanupDB(t))

			cDao := card.NewDao(conn)
			importer := NewImporter(cardset.NewService(cardset.NewDao(conn)), card.NewService(cDao))

			_, err := importer.Import(fromFile(t, "testdata/card/two_translations_create.json"))
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}
			_, err = importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}

			gotCard, _ := cDao.FindUniqueCard("Benalish Hero", "2ED", "4")
			translations, _ := cDao.FindTranslations(gotCard.Id.Int64)
			assertEquals(t, tc.want, translations)
		})
	}
}

func cardSubTypes(t *testing.T) {
	cases := []struct {
		name           string
		fixture        io.Reader
		want           []string
		wantSuperTypes []string
		wantCardTypes  []string
	}{
		{
			name:    "UpdateTypes",
			fixture: fromFile(t, "testdata/type/update.json"),
			want:    []string{"SubType1", "SubType2", "SubType3", "SubType4", "SubType5"},
		},
		{
			name:    "RemoveTypes",
			fixture: fromFile(t, "testdata/type/remove.json"),
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
			_, err = importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}

			gotCard, _ := cDao.FindUniqueCard("Benalish Hero", "2ED", "4")
			types, _ := cDao.FindSubTypes(gotCard.Id.Int64)
			sort.Strings(tc.want)
			sort.Strings(types)

			assertEquals(t, tc.want, types)
		})
	}
}

func cardSuperTypes(t *testing.T) {
	cases := []struct {
		name           string
		fixture        io.Reader
		want           []string
		wantSuperTypes []string
		wantCardTypes  []string
	}{
		{
			name:    "UpdateTypes",
			fixture: fromFile(t, "testdata/type/update.json"),
			want:    []string{"SuperType1"},
		},
		{
			name:    "RemoveTypes",
			fixture: fromFile(t, "testdata/type/remove.json"),
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
			_, err = importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}

			gotCard, _ := cDao.FindUniqueCard("Benalish Hero", "2ED", "4")
			types, _ := cDao.FindSuperTypes(gotCard.Id.Int64)
			sort.Strings(tc.want)
			sort.Strings(types)

			assertEquals(t, tc.want, types)
		})
	}
}

func cardCardTypes(t *testing.T) {
	cases := []struct {
		name           string
		fixture        io.Reader
		want           []string
		wantSuperTypes []string
		wantCardTypes  []string
	}{
		{
			name:    "UpdateTypes",
			fixture: fromFile(t, "testdata/type/update.json"),
			want:    []string{"Type1", "Type2", "Type3", "Type4", "Type5"},
		},
		{
			name:    "RemoveTypes",
			fixture: fromFile(t, "testdata/type/remove.json"),
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
			_, err = importer.Import(tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error during import %v", err)
			}

			gotCard, _ := cDao.FindUniqueCard("Benalish Hero", "2ED", "4")
			types, _ := cDao.FindCardTypes(gotCard.Id.Int64)
			sort.Strings(tc.want)
			sort.Strings(types)

			assertEquals(t, tc.want, types)
		})
	}
}

func runWithDatabase(t *testing.T, runTests func()) {
	ctx := context.Background()
	err := runPostgresContainer(ctx, func(cfg *config.Database) error {
		dbConn, err := postgres.Connect(ctx, cfg)
		if err != nil {
			return err
		}
		defer dbConn.Close()
		conn = dbConn

		cleanupDB = func(t *testing.T) func() {
			return func() {
				cErr := conn.Cleanup()
				if cErr != nil {
					t.Fatalf("failed to cleanup database %v", err)
				}
			}
		}

		runTests()

		return nil
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
	defer postgresC.Terminate(ctx)

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
	return f(dbConfig)
}
