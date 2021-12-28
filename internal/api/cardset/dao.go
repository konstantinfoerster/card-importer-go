package cardset

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
)

type PostgresSetDao struct {
	db *postgres.DBConnection
}

func NewDao(db *postgres.DBConnection) *PostgresSetDao {
	return &PostgresSetDao{
		db: db,
	}
}

func (d *PostgresSetDao) UpdateTranslation(setCode string, t *Translation) error {
	query := "UPDATE card_set_translation SET name = $1 WHERE card_set_code = $2 AND lang_lang = $3"
	ct, err := d.db.Conn.Exec(d.db.Ctx, query, t.Name, t.Lang, setCode)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("no set translations updated but expected to update translation for set with code %s and lang %s", setCode, t.Lang)
	}
	return nil
}

func (d *PostgresSetDao) CreateTranslation(setCode string, t *Translation) error {
	query := "INSERT INTO card_set_translation(name, lang_lang, card_set_code) VALUES($1, $2, $3)"
	_, err := d.db.Conn.Exec(d.db.Ctx, query, t.Name, t.Lang, setCode)
	if err != nil {
		return err
	}
	return nil
}

func (d *PostgresSetDao) DeleteTranslation(setCode string, lang string) error {
	query := `DELETE FROM card_set_translation WHERE card_set_code = $1 AND lang_lang = $2`
	ct, err := d.db.Conn.Exec(d.db.Ctx, query, setCode, lang)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("no set translations deleted but expected to delete translation for set with code %s and lang %s", setCode, lang)
	}
	return nil
}

func (d *PostgresSetDao) FindTranslations(setCode string) ([]*Translation, error) {
	query := `
		SELECT name, lang_lang
		FROM card_set_translation
		WHERE card_set_code = $1
		ORDER BY lang_lang, name`
	rows, err := d.db.Conn.Query(d.db.Ctx, query, setCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Translation
	for rows.Next() {
		t := &Translation{}
		if err := rows.Scan(&t.Name, &t.Lang); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (d *PostgresSetDao) UpdateCardSet(set *CardSet) error {
	query := "UPDATE card_set SET name = $1, type = $2, released = $3, total_count = $4, card_block_id = $5 WHERE code = $6"
	ct, err := d.db.Conn.Exec(d.db.Ctx, query, set.Name, set.Type, set.Released, set.TotalCount, set.Block.Id, set.Code)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("no set updated but expected to update set with code %s", set.Code)
	}
	return nil
}

func (d *PostgresSetDao) CreateCardSet(set *CardSet) error {
	query := `INSERT INTO card_set(code, name, type, released, total_count, card_block_id) 
							VALUES($1, $2, $3, $4, $5, $6)`
	_, err := d.db.Conn.Exec(d.db.Ctx, query, set.Code, set.Name, set.Type, set.Released, set.TotalCount, set.Block.Id)
	if err != nil {
		return err
	}
	return nil
}

func (d *PostgresSetDao) FindCardSetByCode(code string) (*CardSet, error) {
	query := `
		SELECT set.code, set.name, set.type, set.released, set.total_count, block.id, COALESCE(block.block, '')
		FROM card_set AS set
		LEFT join card_block AS block
		ON set.card_block_id = block.id
		WHERE set.code = $1`
	set := &CardSet{Block: CardBlock{}}
	err := d.db.Conn.QueryRow(d.db.Ctx, query, code).
		Scan(&set.Code, &set.Name, &set.Type, &set.Released, &set.TotalCount, &set.Block.Id, &set.Block.Block)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}
	return set, nil
}

func (d *PostgresSetDao) CreateBlock(block string) (*CardBlock, error) {
	b := &CardBlock{
		Block: block,
	}
	err := d.db.Conn.QueryRow(d.db.Ctx, "INSERT INTO card_block(block) VALUES($1) RETURNING id", block).
		Scan(&b.Id)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (d *PostgresSetDao) FindBlockByName(blockName string) (*CardBlock, error) {
	var block CardBlock
	err := d.db.Conn.QueryRow(d.db.Ctx, "SELECT id, block FROM card_block WHERE block = $1", blockName).
		Scan(&block.Id, &block.Block)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}
	return &block, nil
}

func (d *PostgresSetDao) Count() (int, error) {
	var count int
	row := d.db.Conn.QueryRow(d.db.Ctx, "SELECT count(code) FROM card_set")
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
