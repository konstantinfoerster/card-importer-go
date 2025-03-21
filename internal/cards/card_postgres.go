package cards

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
)

var ErrEntryNotFound = errors.New("entry not found")

type PostgresCardDao struct {
	db        *postgres.DBConnection
	subType   TypeDao
	superType TypeDao
	cardType  TypeDao
}

func NewCardDao(db *postgres.DBConnection) *PostgresCardDao {
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
		return f(NewCardDao(txConn))
	})
}

// FindUniqueCard Finds the card with the specified set code and number.
// If no result is found a nil is returned.
func (d *PostgresCardDao) FindUniqueCard(set string, number string) (*Card, error) {
	query := `
		SELECT
			id, name, number, rarity, border, layout, card_set_code
		FROM 
			card
		WHERE 
			card_set_code = $1 AND number = $2`

	var c Card
	err := d.db.Conn.QueryRow(context.TODO(), query, set, number).Scan(&c.ID, &c.Name, &c.Number, &c.Rarity, &c.Border,
		&c.Layout, &c.CardSetCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntryNotFound
		}

		return nil, fmt.Errorf("failed to execute card select %w", err)
	}

	return &c, nil
}

// Paged Returns cards limited by the given size for the given page. The page parameter is one based.
func (d *PostgresCardDao) Paged(page int, size int) ([]Card, error) {
	page--
	if page < 0 {
		page = 0
	}

	offset := page * size
	query := `
		SELECT
			c.id, c.card_set_code AS cardSetCode, c.name, c.number, c.border, c.rarity, c.layout, 
			COALESCE(json_agg(cf), '[]') as faces
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

	rows, err := d.db.Conn.Query(context.TODO(), query, size, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to execute paged card face select %w", err)
	}
	defer rows.Close()

	var result []Card
	for rows.Next() {
		var entry Card
		err := rows.Scan(&entry.ID, &entry.CardSetCode, &entry.Name, &entry.Number, &entry.Border, &entry.Rarity,
			&entry.Layout, &entry.Faces)
		if err != nil {
			return nil, fmt.Errorf("failed to execute card scan after select %w", err)
		}

		result = append(result, entry)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card face result %w", rows.Err())
	}

	return result, nil
}

// CreateCard Creates a new card. Will return an error if the card already exists.
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
	err := d.db.Conn.QueryRow(context.TODO(), query, c.Name, c.Number, c.Rarity, c.Border, c.Layout, c.CardSetCode).
		Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card insert %w", err)
	}
	c.ID = NewPrimaryID(id)

	return nil
}

// UpdateCard Updates an exist card with the given data.
func (d *PostgresCardDao) UpdateCard(c *Card) error {
	query := `
		UPDATE
			card
		SET
			name = $1, number = $2, rarity = $3, border = $4, layout = $5
		WHERE
			id = $6`

	ct, err := d.db.Conn.Exec(context.TODO(), query,
		c.Name, c.Number, c.Rarity, c.Border, c.Layout, c.ID)
	if err != nil {
		return fmt.Errorf("failed to execute card update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d cards updated but expected to update only card with id %d", ra, c.ID.Int64)
	}

	return nil
}

// FindAssignedFaces Returns all card faces for the given card ID.
func (d *PostgresCardDao) FindAssignedFaces(cardID int64) ([]*Face, error) {
	query := `
		SELECT
			id, name, text, flavor_text, type_line, converted_mana_cost, colors, artist,
			hand_modifier, life_modifier, loyalty, mana_cost, multiverse_id, power, toughness
		FROM
			card_face
		WHERE
			card_id = $1
		ORDER BY name`

	rows, err := d.db.Conn.Query(context.TODO(), query, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute card face select %w", err)
	}
	defer rows.Close()

	var result []*Face
	for rows.Next() {
		var entry Face
		err := rows.Scan(
			&entry.ID, &entry.Name, &entry.Text, &entry.FlavorText, &entry.TypeLine, &entry.ConvertedManaCost,
			&entry.Colors, &entry.Artist, &entry.HandModifier, &entry.LifeModifier, &entry.Loyalty, &entry.ManaCost,
			&entry.MultiverseID, &entry.Power, &entry.Toughness)
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

// AddFace Creates a new card face with a reference to the given card ID.
func (d *PostgresCardDao) AddFace(cardID int64, f *Face) error {
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
	err := d.db.Conn.QueryRow(context.TODO(), query,
		f.Name, f.Text, f.FlavorText, f.TypeLine, f.ConvertedManaCost, f.Colors, f.Artist,
		f.HandModifier, f.LifeModifier, f.Loyalty, f.ManaCost, f.MultiverseID, f.Power, f.Toughness, cardID).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card face insert %w", err)
	}
	f.ID = NewPrimaryID(id)

	return nil
}

// UpdateFace Updates an exist card face with the given data.
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

	ct, err := d.db.Conn.Exec(context.TODO(), query,
		f.Name, f.Text, f.FlavorText, f.TypeLine, f.ConvertedManaCost,
		f.Colors, f.Artist, f.HandModifier, f.LifeModifier, f.Loyalty,
		f.ManaCost, f.MultiverseID, f.Power, f.Toughness,
		f.ID)
	if err != nil {
		return fmt.Errorf("failed to execute card face update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d card faces updated but expected to update card face with id %v", ra, f.ID.Int64)
	}

	return nil
}

// DeleteFace Deletes the face with the given face ID and all references to it.
func (d *PostgresCardDao) DeleteFace(faceID int64) error {
	ctx := context.TODO()

	if err := d.deleteAllTranslation(ctx, faceID); err != nil {
		return err
	}
	if err := d.cardType.DeleteAllAssignments(faceID); err != nil {
		return err
	}
	if err := d.subType.DeleteAllAssignments(faceID); err != nil {
		return err
	}
	if err := d.superType.DeleteAllAssignments(faceID); err != nil {
		return err
	}

	query := `
		DELETE FROM
			card_face
		WHERE
			id = $1`

	ct, err := d.db.Conn.Exec(ctx, query, faceID)
	if err != nil {
		return fmt.Errorf("failed to execute delete on card face with id %d %w", faceID, err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d card faces deleted but expected to deleted card face with id %d", ra, faceID)
	}

	return nil
}

// FindTranslations Returns all translations for the given face ID.
func (d *PostgresCardDao) FindTranslations(faceID int64) ([]*FaceTranslation, error) {
	query := `
		SELECT
			name, multiverse_id, text, flavor_text, type_line, lang_lang
		FROM
			card_translation
		WHERE
			face_id = $1
		ORDER BY
			lang_lang, name`

	rows, err := d.db.Conn.Query(context.TODO(), query, faceID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute card translation select %w", err)
	}
	defer rows.Close()

	var result []*FaceTranslation
	for rows.Next() {
		t := &FaceTranslation{}
		if err := rows.Scan(&t.Name, &t.MultiverseID, &t.Text, &t.FlavorText, &t.TypeLine, &t.Lang); err != nil {
			return nil, fmt.Errorf("failed to execute card translation scan after select %w", err)
		}
		result = append(result, t)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card translation result %w", rows.Err())
	}

	return result, nil
}

// AddTranslation Creates a new translation with a reference to the given face ID.
func (d *PostgresCardDao) AddTranslation(faceID int64, t *FaceTranslation) error {
	query := `
		INSERT INTO
			card_translation (
				name, multiverse_id, text, flavor_text, type_line, lang_lang, face_id
			)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	_, err := d.db.Conn.Exec(context.TODO(), query, t.Name, t.MultiverseID, t.Text, t.FlavorText, t.TypeLine, t.Lang,
		faceID)
	if err != nil {
		return fmt.Errorf("failed to execute card_translation insert %w", err)
	}

	return nil
}

// UpdateTranslation Updates an exist face translation with the given data.
func (d *PostgresCardDao) UpdateTranslation(faceID int64, t *FaceTranslation) error {
	query := `
		UPDATE
			card_translation 
		SET
			name = $1, multiverse_id = $2, text = $3, flavor_text = $4, type_line = $5
		WHERE
			face_id = $6 AND lang_lang = $7`

	ct, err := d.db.Conn.Exec(context.TODO(), query,
		t.Name, t.MultiverseID, t.Text, t.FlavorText, t.TypeLine, faceID, t.Lang)
	if err != nil {
		return fmt.Errorf("failed to execute face translation update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d face translations updated but expected to update translation for face with "+
			"id %d and lang %s", ra, faceID, t.Lang)
	}

	return nil
}

func (d *PostgresCardDao) deleteAllTranslation(ctx context.Context, faceID int64) error {
	query := `
		DELETE FROM
			card_translation
		WHERE
			face_id = $1`

	_, err := d.db.Conn.Exec(ctx, query, faceID)
	if err != nil {
		return fmt.Errorf("failed to execute delete on face translation with id %d %w", faceID, err)
	}

	return nil
}

// DeleteTranslation Deletes a language specific face translation.
func (d *PostgresCardDao) DeleteTranslation(faceID int64, lang string) error {
	query := `
		DELETE FROM
			card_translation
		WHERE
			face_id = $1 AND lang_lang = $2`
	ct, err := d.db.Conn.Exec(context.TODO(), query, faceID, lang)
	if err != nil {
		return fmt.Errorf("failed to execute delete on face translation with id %d %w", faceID, err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d face translations deleted but expected to deleted translation for face with "+
			"id %d and lang %s", ra, faceID, lang)
	}

	return nil
}

// FindAssignedSubTypes Returns all subtypes that refer to the given face ID.
func (d *PostgresCardDao) FindAssignedSubTypes(faceID int64) ([]string, error) {
	return findTypes(d.subType, faceID)
}

// FindAssignedSuperTypes Returns all supertypes that refer to the given face ID.
func (d *PostgresCardDao) FindAssignedSuperTypes(faceID int64) ([]string, error) {
	return findTypes(d.superType, faceID)
}

// FindAssignedCardTypes Returns all cardtype that refer to the given face ID.
func (d *PostgresCardDao) FindAssignedCardTypes(faceID int64) ([]string, error) {
	return findTypes(d.cardType, faceID)
}

func findTypes(dao TypeDao, faceID int64) ([]string, error) {
	assignments, err := dao.FindAssignments(faceID)
	if err != nil {
		return nil, err
	}

	var types []string
	for _, t := range assignments {
		types = append(types, t.Name)
	}

	return types, nil
}

// Count Returns the amount of all cards.
func (d *PostgresCardDao) Count() (int, error) {
	row := d.db.Conn.QueryRow(context.TODO(), "SELECT count(id) FROM card")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute card count %w", err)
	}

	return count, nil
}

// IsImagePresent Checks if the card image with the specified id and language exist.
func (d *PostgresCardDao) IsImagePresent(ctx context.Context, faceID int64, lang string) (bool, error) {
	query := `
		SELECT
			count(*) > 0
		FROM 
			card_image
		WHERE
			lang_lang = $1 AND face_id = $2`
	var isPresent bool
	err := d.db.Conn.QueryRow(ctx, query, lang, faceID).Scan(&isPresent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("failed to execute count on card_image %w", err)
	}

	return isPresent, nil
}

// CountImages Returns the amount of all card images.
func (d *PostgresCardDao) CountImages() (int, error) {
	row := d.db.Conn.QueryRow(context.TODO(), "SELECT count(id) FROM card_image")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute card_image count %w", err)
	}

	return count, nil
}

// AddImage Creates a new card image.
func (d *PostgresCardDao) AddImage(ctx context.Context, img *Image) error {
	query := `
		INSERT INTO
			card_image (
				image_path, lang_lang, card_id, face_id, mime_type, 
                phash1, phash2, phash3, phash4
			) 
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING
			id`
	var id int64
	err := d.db.Conn.QueryRow(ctx, query,
		img.ImagePath, img.Lang, img.CardID, img.FaceID, img.MimeType,
		fmt.Sprintf("%064b", img.PHash1),
		fmt.Sprintf("%064b", img.PHash2),
		fmt.Sprintf("%064b", img.PHash3),
		fmt.Sprintf("%064b", img.PHash4),
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to execute card insert %w", err)
	}
	img.ID = NewPrimaryID(id)

	return nil
}

func (d *PostgresCardDao) GetImages() ([]*Image, error) {
	query := `
		SELECT
			id, image_path, card_id, face_id, mime_type, phash1, phash2,
            phash3, phash4, lang_lang
		FROM
			card_image
        `
	rows, err := d.db.Conn.Query(context.TODO(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute select on card_image %w", err)
	}
	defer rows.Close()

	var result []*Image
	for rows.Next() {
		var img Image
		var phash1 pgtype.Varbit
		var phash2 pgtype.Varbit
		var phash3 pgtype.Varbit
		var phash4 pgtype.Varbit
		rErr := rows.Scan(&img.ID, &img.ImagePath, &img.CardID,
			&img.FaceID, &img.MimeType, &phash1, &phash2, &phash3, &phash4, &img.Lang)
		if rErr != nil {
			return nil, fmt.Errorf("failed to execute select on card_image %w", rErr)
		}

		img.PHash1 = binary.BigEndian.Uint64(phash1.Bytes)
		img.PHash2 = binary.BigEndian.Uint64(phash2.Bytes)
		img.PHash3 = binary.BigEndian.Uint64(phash3.Bytes)
		img.PHash4 = binary.BigEndian.Uint64(phash4.Bytes)

		result = append(result, &img)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read card image result %w", rows.Err())
	}

	return result, nil
}

func (d *PostgresCardDao) UpdateHashes(
	id int64, phash1 uint64, phash2 uint64, phash3 uint64, phash4 uint64) error {
	nPhash1 := fmt.Sprintf("%064b", phash1)
	maxLength := 64
	if len(nPhash1) != maxLength {
		return fmt.Errorf("phash1 %s must have a length of 64, but got %d", nPhash1, len(nPhash1))
	}
	nPhash2 := fmt.Sprintf("%064b", phash2)
	if len(nPhash2) != maxLength {
		return fmt.Errorf("phash2 %s must have a length of 64, but got %d", nPhash2, len(nPhash2))
	}
	nPhash3 := fmt.Sprintf("%064b", phash3)
	if len(nPhash3) != maxLength {
		return fmt.Errorf("phash3 %s must have a length of 64, but got %d", nPhash3, len(nPhash3))
	}
	nPhash4 := fmt.Sprintf("%064b", phash4)
	if len(nPhash4) != maxLength {
		return fmt.Errorf("phash4 %s must have a length of 64, but got %d", nPhash4, len(nPhash4))
	}

	query := `
		UPDATE
			card_image 
		SET
			phash1=$2,
			phash2=$3,
			phash3=$4,
			phash4=$5
        WHERE
			id = $1`

	ct, err := d.db.Conn.Exec(context.TODO(), query, id,
		nPhash1, nPhash2, nPhash3, nPhash4)
	if err != nil {
		return fmt.Errorf("failed to execute card image update %w", err)
	}
	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d card image updated but expected to update card image with "+
			"id %d", ra, id)
	}

	return nil
}
