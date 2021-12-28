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

// TODO make this more maintainable

// FindUniqueCard finds the card with the specified name, set code and number.
// If no result is found a nil card is returned
func (d *PostgresCardDao) FindUniqueCard(name string, set string, number string) (*Card, error) {
	query := `SELECT id, artist, border, converted_mana_cost, colors, name, text, flavor_text, layout, 
					hand_modifier, life_modifier, loyalty, mana_cost, multiverse_id, power, toughness, 
					rarity, number, full_type, card_set_code
			  FROM card WHERE name = $1 AND card_set_code = $2 AND number = $3
			  ORDER BY card_set_code, name`
	var colors string
	var c Card
	err := d.db.Conn.QueryRow(d.db.Ctx, query, name, set, number).Scan(
		&c.Id, &c.Artist, &c.Border, &c.ConvertedManaCost, &colors, &c.Name, &c.Text, &c.FlavorText, &c.Layout,
		&c.HandModifier, &c.LifeModifier, &c.Loyalty, &c.ManaCost, &c.MultiverseId, &c.Power, &c.Toughness,
		&c.Rarity, &c.Number, &c.FullType, &c.CardSetCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	c.Colors = strings.Split(colors, ",")
	return &c, nil
}

// TODO make this more maintainable
func (d *PostgresCardDao) CreateCard(c *Card) error {
	colors := strings.Join(c.Colors, ",")
	query := `INSERT INTO 
				card(artist, border, converted_mana_cost, colors, name, text, flavor_text, layout, 
					hand_modifier, life_modifier, loyalty, mana_cost, multiverse_id, power, toughness, 
					rarity, number, full_type, card_set_code) 
				VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
				RETURNING id`
	var id int64
	err := d.db.Conn.QueryRow(d.db.Ctx, query,
		c.Artist, c.Border, c.ConvertedManaCost, colors, c.Name, c.Text, c.FlavorText, c.Layout,
		c.HandModifier, c.LifeModifier, c.Loyalty, c.ManaCost, c.MultiverseId, c.Power, c.Toughness,
		c.Rarity, c.Number, c.FullType, c.CardSetCode).Scan(&id)
	if err != nil {
		return err
	}
	c.Id = sql.NullInt64{Int64: id, Valid: true}
	return nil
}

// TODO make this more maintainable
// TODO remove name and set code from value list??
func (d *PostgresCardDao) UpdateCard(c *Card) error {
	colors := strings.Join(c.Colors, ",")
	query := `UPDATE card
				SET artist = $1, border = $2, converted_mana_cost = $3, colors = $4, name = $5,
					text = $6, flavor_text = $7, layout = $8, hand_modifier = $9, life_modifier = $10,
					loyalty = $11, mana_cost = $12, multiverse_id = $13, power = $14, toughness = $15, 
					rarity = $16, number = $17, full_type = $18, card_set_code = $19
				 WHERE id = $20`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query,
		c.Artist, c.Border, c.ConvertedManaCost, colors, c.Name, c.Text, c.FlavorText, c.Layout,
		c.HandModifier, c.LifeModifier, c.Loyalty, c.ManaCost, c.MultiverseId, c.Power, c.Toughness,
		c.Rarity, c.Number, c.FullType, c.CardSetCode, c.Id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("no cards updated but expected to update card with id %d", c.Id.Int64)
	}
	return nil
}

func (d *PostgresCardDao) FindTranslations(cardId int64) ([]*Translation, error) {
	query := `
		SELECT name, multiverse_id, text, flavor_text, full_type, lang_lang
		FROM card_translation
		WHERE card_id = $1
		ORDER BY lang_lang, name`
	rows, err := d.db.Conn.Query(d.db.Ctx, query, cardId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Translation
	for rows.Next() {
		t := &Translation{}
		if err := rows.Scan(&t.Name, &t.MultiverseId, &t.Text, &t.FlavorText, &t.FullType, &t.Lang); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (d *PostgresCardDao) CreateTranslation(cardId int64, t *Translation) error {
	query := `INSERT INTO card_translation(
				name, multiverse_id, text, flavor_text, full_type, lang_lang, card_id
			   ) VALUES($1, $2, $3, $4, $5, $6, $7)`
	_, err := d.db.Conn.Exec(d.db.Ctx, query, t.Name, t.MultiverseId, t.Text, t.FlavorText, t.FullType, t.Lang, cardId)
	if err != nil {
		return err
	}
	return nil
}

func (d *PostgresCardDao) UpdateTranslation(cardId int64, t *Translation) error {
	query := `UPDATE card_translation 
				SET name = $1, multiverse_id = $2, text = $3, flavor_text = $4, full_type = $5
			    WHERE card_id = $6 AND lang_lang = $7`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query,
		t.Name, t.MultiverseId, t.Text, t.FlavorText, t.FullType, cardId, t.Lang)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("no card translations updated but expected to update translation for card with id %d and lang %s", cardId, t.Lang)
	}
	return nil
}

func (d *PostgresCardDao) DeleteTranslation(cardId int64, lang string) error {
	query := `DELETE FROM card_translation WHERE card_id = $1 AND lang_lang = $2`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query, cardId, lang)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("no card translations deleted but expected to deleted translation for card with id %d and lang %s", cardId, lang)
	}
	return nil
}

func (d *PostgresCardDao) FindSubTypes(cardId int64) ([]string, error) {
	return findTypes(d.subType, cardId)
}

func (d *PostgresCardDao) FindSuperTypes(cardId int64) ([]string, error) {
	return findTypes(d.superType, cardId)
}

func (d *PostgresCardDao) FindCardTypes(cardId int64) ([]string, error) {
	return findTypes(d.cardType, cardId)
}

func findTypes(dao TypeDao, cardId int64) ([]string, error) {
	assignments, err := dao.FindAssignments(cardId)
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
		return 0, err
	}
	return count, nil
}
