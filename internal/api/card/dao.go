package card

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"strings"
)

type PostgresCardDao struct {
	db        *postgres.DBConnection
	subType   TypeDao
	superType TypeDao
	cardType  TypeDao
}

func NewDao(db *postgres.DBConnection) *PostgresCardDao {
	return &PostgresCardDao{
		db:        db,
		subType:   NewSubTypeDao(db),
		superType: NewSuperTypeDao(db),
		cardType:  NewCardTypeDao(db),
	}
}

func (d *PostgresCardDao) withTransaction(f func(txDao *PostgresCardDao) error) error {
	// create a new dao instance with a transactional connection
	return d.db.WithTransaction(func(txConn *postgres.DBConnection) error {
		return f(NewDao(txConn))
	})
}

// FindUniqueCard finds the card with the specified set code and number.
// If no result is found a nil card is returned
func (d *PostgresCardDao) FindUniqueCard(set string, number string) (*Card, error) {
	query := `
		SELECT
			id, name, number, rarity, border, layout, card_set_code
		FROM 
			card
		WHERE 
			card_set_code = $1 AND number = $2`
	var c Card
	err := d.db.Conn.QueryRow(d.db.Ctx, query, set, number).Scan(&c.Id, &c.Name, &c.Number, &c.Rarity, &c.Border, &c.Layout, &c.CardSetCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to execute card select %w", err)
	}
	return &c, nil
}

func (d *PostgresCardDao) CreateCard(c *Card) error {
	query := `
		INSERT INTO
			card (
				name, number, rarity, border, layout, card_set_code
			) 
		VALUES (
			$1, $2, $3, $4, $5, $6
		)
		RETURNING
			id`
	var id int64
	err := d.db.Conn.QueryRow(d.db.Ctx, query, c.Name, c.Number, c.Rarity, c.Border, c.Layout, c.CardSetCode).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card insert %w", err)
	}
	c.Id = sql.NullInt64{Int64: id, Valid: true}
	return nil
}

func (d *PostgresCardDao) UpdateCard(c *Card) error {
	query := `
		UPDATE
			card
		SET
			name = $1, number = $2, rarity = $3, border = $4, layout = $5
		WHERE
			id = $6`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query,
		c.Name, c.Number, c.Rarity, c.Border, c.Layout, c.Id)
	if err != nil {
		return fmt.Errorf("failed to execute card update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d cards updated but expected to update only card with id %d", ra, c.Id.Int64)
	}
	return nil
}

func (d *PostgresCardDao) FindAssignedFaces(cardId int64) ([]*Face, error) {
	query := `
		SELECT
			id, name, text, flavor_text, type_line, converted_mana_cost, colors, artist,
			hand_modifier, life_modifier, loyalty, mana_cost, multiverse_id, power, toughness
		FROM
			card_face
		WHERE
			card_id = $1
		ORDER BY name`
	rows, err := d.db.Conn.Query(d.db.Ctx, query, cardId)
	if err != nil {
		return nil, fmt.Errorf("failed to execute card face select %w", err)
	}

	defer rows.Close()
	var result []*Face
	for rows.Next() {
		var entry Face
		var colors string
		err := rows.Scan(
			&entry.Id, &entry.Name, &entry.Text, &entry.FlavorText, &entry.TypeLine, &entry.ConvertedManaCost, &colors, &entry.Artist,
			&entry.HandModifier, &entry.LifeModifier, &entry.Loyalty, &entry.ManaCost, &entry.MultiverseId, &entry.Power, &entry.Toughness)
		if err != nil {
			return nil, fmt.Errorf("failed to execute face scan after select %w", err)
		}
		if len(colors) != 0 {
			entry.Colors = strings.Split(colors, ",")
		}

		result = append(result, &entry)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card face result %w", rows.Err())
	}
	return result, nil
}

func (d *PostgresCardDao) AddFace(cardId int64, f *Face) error {
	colors := strings.Join(f.Colors, ",")
	query := `
		INSERT INTO 
			card_face (
				name, text, flavor_text, type_line, converted_mana_cost, colors, artist,
				hand_modifier, life_modifier, loyalty, mana_cost, multiverse_id, power, toughness, card_id
			) 
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING
			id`
	var id int64
	err := d.db.Conn.QueryRow(d.db.Ctx, query,
		f.Name, f.Text, f.FlavorText, f.TypeLine, f.ConvertedManaCost, colors, f.Artist,
		f.HandModifier, f.LifeModifier, f.Loyalty, f.ManaCost, f.MultiverseId, f.Power, f.Toughness, cardId).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card face insert %w", err)
	}
	f.Id = sql.NullInt64{Int64: id, Valid: true}
	return nil
}

func (d *PostgresCardDao) UpdateFace(f *Face) error {
	colors := strings.Join(f.Colors, ",")
	query := `
		UPDATE
			card_face
		SET
			name = $1, text = $2, flavor_text = $3, type_line = $4, converted_mana_cost = $5,
			colors = $6, artist = $7, hand_modifier = $8, life_modifier = $9, loyalty = $10, 
			mana_cost = $11, multiverse_id = $12, power = $13, toughness = $14
		WHERE
			id = $15`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query,
		f.Name, f.Text, f.FlavorText, f.TypeLine, f.ConvertedManaCost,
		colors, f.Artist, f.HandModifier, f.LifeModifier, f.Loyalty,
		f.ManaCost, f.MultiverseId, f.Power, f.Toughness,
		f.Id)
	if err != nil {
		return fmt.Errorf("failed to execute card face update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d card faces updated but expected to update card face with id %v", ra, f.Id.Int64)
	}
	return nil
}

func (d *PostgresCardDao) DeleteFace(faceId int64) error {
	query := `
		DELETE FROM
			card_face
		WHERE
			id = $1`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query, faceId)
	if err != nil {
		return fmt.Errorf("failed to execute card face delete %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d card faces deleted but expected to deleted card face with id %d", ra, faceId)
	}
	return nil
}

func (d *PostgresCardDao) FindTranslations(faceId int64) ([]*Translation, error) {
	query := `
		SELECT
			name, multiverse_id, text, flavor_text, type_line, lang_lang
		FROM
			card_translation
		WHERE
			face_id = $1
		ORDER BY
			lang_lang, name`
	rows, err := d.db.Conn.Query(d.db.Ctx, query, faceId)
	if err != nil {
		return nil, fmt.Errorf("failed to execute card translation select %w", err)
	}
	defer rows.Close()

	var result []*Translation
	for rows.Next() {
		t := &Translation{}
		if err := rows.Scan(&t.Name, &t.MultiverseId, &t.Text, &t.FlavorText, &t.TypeLine, &t.Lang); err != nil {
			return nil, fmt.Errorf("failed to execute card translation scan after select %w", err)
		}
		result = append(result, t)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card translation result %w", rows.Err())
	}
	return result, nil
}

func (d *PostgresCardDao) AddTranslation(faceId int64, t *Translation) error {
	query := `
		INSERT INTO
			card_translation (
				name, multiverse_id, text, flavor_text, type_line, lang_lang, face_id
			)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`
	_, err := d.db.Conn.Exec(d.db.Ctx, query, t.Name, t.MultiverseId, t.Text, t.FlavorText, t.TypeLine, t.Lang, faceId)
	if err != nil {
		return fmt.Errorf("failed to execute card_translation insert %w", err)
	}
	return nil
}

func (d *PostgresCardDao) UpdateTranslation(faceId int64, t *Translation) error {
	query := `
		UPDATE
			card_translation 
		SET
			name = $1, multiverse_id = $2, text = $3, flavor_text = $4, type_line = $5
		WHERE
			face_id = $6 AND lang_lang = $7`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query,
		t.Name, t.MultiverseId, t.Text, t.FlavorText, t.TypeLine, faceId, t.Lang)
	if err != nil {
		return fmt.Errorf("failed to execute face translation update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d face translations updated but expected to update translation for face with id %d and lang %s", ra, faceId, t.Lang)
	}
	return nil
}

func (d *PostgresCardDao) DeleteAllTranslation(faceId int64) error {
	query := `
		DELETE FROM
			card_translation
		WHERE
			face_id = $1`
	_, err := d.db.Conn.Exec(d.db.Ctx, query, faceId)
	if err != nil {
		return fmt.Errorf("failed to execute face translation delete %w", err)
	}
	return nil
}

func (d *PostgresCardDao) DeleteTranslation(faceId int64, lang string) error {
	query := `
		DELETE FROM
			card_translation
		WHERE
			face_id = $1 AND lang_lang = $2`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query, faceId, lang)
	if err != nil {
		return fmt.Errorf("failed to execute face translation delete %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d face translations deleted but expected to deleted translation for face with id %d and lang %s", ra, faceId, lang)
	}
	return nil
}

func (d *PostgresCardDao) FindAssignedSubTypes(faceId int64) ([]string, error) {
	return findTypes(d.subType, faceId)
}

func (d *PostgresCardDao) FindAssignedSuperTypes(faceId int64) ([]string, error) {
	return findTypes(d.superType, faceId)
}

func (d *PostgresCardDao) FindAssignedCardTypes(faceId int64) ([]string, error) {
	return findTypes(d.cardType, faceId)
}

func findTypes(dao TypeDao, faceId int64) ([]string, error) {
	assignments, err := dao.FindAssignments(faceId)
	if err != nil {
		return nil, err
	}

	var types []string
	for _, t := range assignments {
		types = append(types, t.Name)
	}
	return types, nil
}

func (d *PostgresCardDao) Count() (int, error) {
	row := d.db.Conn.QueryRow(d.db.Ctx, "SELECT count(id) FROM card")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute card count %w", err)
	}
	return count, nil
}
