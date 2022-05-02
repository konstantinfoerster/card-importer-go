package card

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
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
// If no result is found a nil is returned
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

func (d *PostgresCardDao) Paged(page int, size int) ([]*Card, error) {
	page = page - 1
	if page < 0 {
		page = 0
	}
	offset := page * size
	query := `
		SELECT
			c.id, c.card_set_code AS cardSetCode, c.name, c.number, c.border, c.rarity, c.layout, COALESCE(json_agg(cf), '[]') as faces
		FROM
			card AS c
		LEFT JOIN
			card_face AS cf
		ON
			c.id = cf.card_id
		GROUP BY
			c.id
		ORDER BY
			cardSetCode
		LIMIT $1
		OFFSET $2`
	rows, err := d.db.Conn.Query(d.db.Ctx, query, size, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []*Card
	for rows.Next() {
		var entry Card
		err := rows.Scan(&entry.Id, &entry.CardSetCode, &entry.Name, &entry.Number, &entry.Border, &entry.Rarity, &entry.Layout, &entry.Faces)
		if err != nil {
			return nil, fmt.Errorf("failed to execute card scan after select %w", err)
		}

		result = append(result, &entry)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card face result %w", rows.Err())
	}
	return result, nil
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
	c.Id = NewPrimaryId(id)
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
		err := rows.Scan(
			&entry.Id, &entry.Name, &entry.Text, &entry.FlavorText, &entry.TypeLine, &entry.ConvertedManaCost, &entry.Colors, &entry.Artist,
			&entry.HandModifier, &entry.LifeModifier, &entry.Loyalty, &entry.ManaCost, &entry.MultiverseId, &entry.Power, &entry.Toughness)
		if err != nil {
			return nil, fmt.Errorf("failed to execute face scan after select %w", err)
		}

		result = append(result, &entry)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card face result %w", rows.Err())
	}
	return result, nil
}

func (d *PostgresCardDao) AddFace(cardId int64, f *Face) error {
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
		f.Name, f.Text, f.FlavorText, f.TypeLine, f.ConvertedManaCost, f.Colors, f.Artist,
		f.HandModifier, f.LifeModifier, f.Loyalty, f.ManaCost, f.MultiverseId, f.Power, f.Toughness, cardId).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card face insert %w", err)
	}
	f.Id = NewPrimaryId(id)
	return nil
}

func (d *PostgresCardDao) UpdateFace(f *Face) error {
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
		f.Colors, f.Artist, f.HandModifier, f.LifeModifier, f.Loyalty,
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

// IsImagePresent checks if the card image with the specified card id and language
func (d *PostgresCardDao) IsImagePresent(cardId int64, lang string) (bool, error) {
	query := `
		SELECT
			count(*) > 0
		FROM 
			card_image
		WHERE
			lang_lang = $1 AND card_id = $2`
	var isPresent bool
	err := d.db.Conn.QueryRow(d.db.Ctx, query, lang, cardId).Scan(&isPresent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("failed to execute count on card_image %w", err)
	}
	return isPresent, nil
}

func (d *PostgresCardDao) CountImages() (int, error) {
	row := d.db.Conn.QueryRow(d.db.Ctx, "SELECT count(id) FROM card_image")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute card_image count %w", err)
	}
	return count, nil
}

func (d *PostgresCardDao) AddImage(img *CardImage) error {
	query := `
		INSERT INTO
			card_image (
				image_path, lang_lang, card_id, face_id, mime_type
			) 
		VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING
			id`
	var id int64
	err := d.db.Conn.QueryRow(d.db.Ctx, query, img.ImagePath, img.Lang, img.CardId, img.FaceId, img.MimeType).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card insert %w", err)
	}
	img.Id = NewPrimaryId(id)
	return nil
}
