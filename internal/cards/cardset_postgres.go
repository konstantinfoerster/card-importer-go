package cards

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
)

type PostgresSetDao struct {
	db *postgres.DBConnection
}

func NewSetDao(db *postgres.DBConnection) *PostgresSetDao {
	return &PostgresSetDao{
		db: db,
	}
}

func (d *PostgresSetDao) UpdateTranslation(setCode string, t *SetTranslation) error {
	query := `
		UPDATE
			card_set_translation
		SET
			name = $1
		WHERE
			lang_lang = $2 AND card_set_code = $3`

	ct, err := d.db.Conn.Exec(context.TODO(), query, t.Name, t.Lang, setCode)
	if err != nil {
		return fmt.Errorf("failed to update set translation %w", err)
	}

	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d translations updated but expected to update translation for set with code %s and lang %s",
			ra, setCode, t.Lang)
	}

	return nil
}

func (d *PostgresSetDao) CreateTranslation(setCode string, t *SetTranslation) error {
	query := `
			INSERT INTO
				card_set_translation(
					name, lang_lang, card_set_code
				)
			VALUES
				($1, $2, $3)`

	_, err := d.db.Conn.Exec(context.TODO(), query, t.Name, t.Lang, setCode)
	if err != nil {
		return fmt.Errorf("failed to insert set translation %w", err)
	}

	return nil
}

func (d *PostgresSetDao) DeleteTranslation(setCode string, lang string) error {
	query := `
			DELETE FROM
				card_set_translation
			WHERE
				lang_lang = $1 AND card_set_code = $2`

	ct, err := d.db.Conn.Exec(context.TODO(), query, lang, setCode)
	if err != nil {
		return fmt.Errorf("failed to delete set translation %w", err)
	}

	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d set translations deleted but expected to delete translation for set with "+
			"code %s and lang %s", ra, setCode, lang)
	}

	return nil
}

func (d *PostgresSetDao) FindTranslations(setCode string) ([]*SetTranslation, error) {
	query := `
		SELECT
			name, lang_lang
		FROM
			card_set_translation
		WHERE
			card_set_code = $1
		ORDER BY
			lang_lang, name`

	rows, err := d.db.Conn.Query(context.TODO(), query, setCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*SetTranslation
	for rows.Next() {
		t := &SetTranslation{}
		if err := rows.Scan(&t.Name, &t.Lang); err != nil {
			return nil, fmt.Errorf("failed to scan after set translation select %w", err)
		}
		result = append(result, t)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read row after set translation select %w", rows.Err())
	}

	return result, nil
}

func (d *PostgresSetDao) UpdateCardSet(set *CardSet) error {
	query := `
		UPDATE
			card_set
		SET
			name = $1, type = $2, released = $3, total_count = $4, card_block_id = $5
		WHERE
			code = $6`

	ct, err := d.db.Conn.Exec(context.TODO(), query, set.Name, set.Type, set.Released, set.TotalCount, set.Block.ID,
		set.Code)
	if err != nil {
		return fmt.Errorf("failed to update set %w", err)
	}

	ra := ct.RowsAffected()
	if ra != 1 {
		return fmt.Errorf("%d sets updated but expected to update set with code %s", ra, set.Code)
	}

	return nil
}

func (d *PostgresSetDao) CreateCardSet(set *CardSet) error {
	query := `
		INSERT INTO
			card_set (
				code, name, type, released, total_count, card_block_id
			) 
		VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	_, err := d.db.Conn.Exec(context.TODO(), query, set.Code, set.Name, set.Type, set.Released, set.TotalCount,
		set.Block.ID)
	if err != nil {
		return fmt.Errorf("failed to create set %w", err)
	}

	return nil
}

func (d *PostgresSetDao) FindCardSetByCode(code string) (*CardSet, error) {
	query := `
		SELECT
			cs.code, cs.name, cs.type, cs.released, cs.total_count, cb.id, COALESCE(cb.block, '')
		FROM
			card_set AS cs
		LEFT join
			card_block AS cb
		ON
			cs.card_block_id = cb.id
		WHERE
			cs.code = $1`

	set := &CardSet{Block: CardBlock{}}
	err := d.db.Conn.QueryRow(context.TODO(), query, code).
		Scan(&set.Code, &set.Name, &set.Type, &set.Released, &set.TotalCount, &set.Block.ID, &set.Block.Block)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntryNotFound
		}

		return nil, fmt.Errorf("failed to select set by code %s %w", code, err)
	}

	return set, nil
}

func (d *PostgresSetDao) CreateBlock(block string) (*CardBlock, error) {
	query := `
		INSERT INTO
			card_block(block)
		VALUES
			($1)
		RETURNING
			id`

	b := &CardBlock{
		Block: block,
	}
	err := d.db.Conn.QueryRow(context.TODO(), query, block).Scan(&b.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert block %w", err)
	}

	return b, nil
}

func (d *PostgresSetDao) FindBlockByName(blockName string) (*CardBlock, error) {
	query := `
		SELECT
			id, block
		FROM
			card_block
		WHERE
			block = $1`

	var block CardBlock
	err := d.db.Conn.QueryRow(context.TODO(), query, blockName).Scan(&block.ID, &block.Block)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntryNotFound
		}

		return nil, fmt.Errorf("failed to select block by name %s %w", blockName, err)
	}

	return &block, nil
}

func (d *PostgresSetDao) Count() (int, error) {
	query := `
		SELECT
			count(code)
		FROM
			card_set`

	row := d.db.Conn.QueryRow(context.TODO(), query)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count sets %w", err)
	}

	return count, nil
}
