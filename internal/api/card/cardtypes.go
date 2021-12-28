package card

import (
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"strings"
)

type TypeDao interface {
	Create(name string) (*CharacteristicType, error)
	Find(names ...string) ([]*CharacteristicType, error)
	AssignToCard(cardId int64, typeId int64) error
	FindAssignments(cardId int64) ([]*CharacteristicType, error)
	DeleteAssignments(cardId int64, subTypeIds ...int64) error
}

func NewSubTypeDao(db *postgres.DBConnection) TypeDao {
	return &CharacteristicDao{db: db, tableName: "sub_type", joinTable: "card_sub_type"}
}

func NewSuperTypeDao(db *postgres.DBConnection) TypeDao {
	return &CharacteristicDao{db: db, tableName: "super_type", joinTable: "card_super_type"}
}
func NewCardTypeDao(db *postgres.DBConnection) TypeDao {
	return &CharacteristicDao{db: db, tableName: "card_type", joinTable: "card_card_type"}
}

type CharacteristicDao struct {
	db        *postgres.DBConnection
	tableName string
	joinTable string
}

func newEntity(id sql.NullInt64, name string) *CharacteristicType {
	return &CharacteristicType{Id: id, Name: name}
}

func (d *CharacteristicDao) Create(name string) (*CharacteristicType, error) {
	var id int64
	err := d.db.Conn.QueryRow(d.db.Ctx, "INSERT INTO "+d.tableName+"(name) VALUES($1) RETURNING id", name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return newEntity(sql.NullInt64{Int64: id, Valid: true}, name), nil
}

func (d *CharacteristicDao) Find(names ...string) ([]*CharacteristicType, error) {
	if len(names) == 0 {
		return nil, nil
	}

	var result []*CharacteristicType

	var params []interface{}
	var inPart strings.Builder
	for i, name := range names {
		if i > 0 {
			inPart.WriteString(", ")
		}
		params = append(params, name)
		inPart.WriteString(fmt.Sprintf("$%d", len(params)))
	}

	wherePart := "name in (" + inPart.String() + ")"
	if len(params) == 1 {
		wherePart = "name = $1"
	}
	rows, err := d.db.Conn.Query(d.db.Ctx, "SELECT id, name FROM "+d.tableName+" WHERE "+wherePart+" ORDER BY name", params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry CharacteristicType
		err := rows.Scan(&entry.Id, &entry.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, &entry)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return result, nil
}

func (d *CharacteristicDao) AssignToCard(cardId int64, typeId int64) error {
	_, err := d.db.Conn.Exec(d.db.Ctx, "INSERT INTO card_"+d.tableName+"(card_id, type_id) VALUES($1, $2)", cardId, typeId)
	if err != nil {
		return err
	}
	return nil
}
func (d *CharacteristicDao) FindAssignments(cardId int64) ([]*CharacteristicType, error) {
	rows, err := d.db.Conn.Query(d.db.Ctx, `
			SELECT t.id, t.name 
			FROM `+d.tableName+` t JOIN `+d.joinTable+` ct ON t.id = ct.type_id
			WHERE ct.card_id = $1
			ORDER BY t.name`, cardId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*CharacteristicType
	for rows.Next() {
		var entry CharacteristicType
		err := rows.Scan(&entry.Id, &entry.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, &entry)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return result, nil
}

func (d *CharacteristicDao) DeleteAssignments(cardId int64, typeIds ...int64) error {
	if len(typeIds) == 0 {
		return nil
	}
	var params []interface{}
	// param $1 is the card id
	params = append(params, cardId)

	var inPart strings.Builder
	for i, id := range typeIds {
		if i > 0 {
			inPart.WriteString(", ")
		}
		params = append(params, id)
		inPart.WriteString(fmt.Sprintf("$%d", len(params)))
	}

	inPart.WriteString(", ")
	inPart.WriteString(fmt.Sprintf("$%d", len(typeIds)+1)) // additional one because of card_id

	query := "DELETE FROM " + d.joinTable + " WHERE card_id = $1 AND type_id IN (" + inPart.String() + ")"

	ct, err := d.db.Conn.Exec(d.db.Ctx, query, params...)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != int64(len(typeIds)) {
		return fmt.Errorf("expected to deleted %d assigned types but deleted %d from card with id %d, %s", len(typeIds), ct.RowsAffected(), cardId, query)
	}
	return nil
}
