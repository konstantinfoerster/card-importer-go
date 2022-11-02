package card

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/postgres"
	"strings"
)

type TypeDao interface {
	Create(name string) (*CharacteristicType, error)
	Find(names ...string) ([]*CharacteristicType, error)
	AssignToFace(faceId int64, typeId int64) error
	FindAssignments(faceId int64) ([]*CharacteristicType, error)
	DeleteAssignments(faceId int64, subTypeIds ...int64) error
	DeleteAllAssignments(faceId int64) error
}

func NewSubTypeDao(db *postgres.DBConnection) TypeDao {
	return &CharacteristicDao{db: db, tableName: "sub_type", joinTable: "face_sub_type"}
}

func NewSuperTypeDao(db *postgres.DBConnection) TypeDao {
	return &CharacteristicDao{db: db, tableName: "super_type", joinTable: "face_super_type"}
}
func NewCardTypeDao(db *postgres.DBConnection) TypeDao {
	return &CharacteristicDao{db: db, tableName: "card_type", joinTable: "face_card_type"}
}

type CharacteristicDao struct {
	db        *postgres.DBConnection
	tableName string
	joinTable string
}

func newEntity(id PrimaryId, name string) *CharacteristicType {
	return &CharacteristicType{Id: id, Name: name}
}

func (d *CharacteristicDao) Create(name string) (*CharacteristicType, error) {
	query := fmt.Sprintf(`
		INSERT INTO
			%s(name)
		VALUES
			($1)
	    ON CONFLICT (name) DO UPDATE
		SET 
			name = $1
		RETURNING
			id`, d.tableName)
	var id int64
	err := d.db.Conn.QueryRow(d.db.Ctx, query, name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return newEntity(NewPrimaryId(id), name), nil
}

func (d *CharacteristicDao) Find(names ...string) ([]*CharacteristicType, error) {
	if len(names) == 0 {
		return []*CharacteristicType{}, nil
	}

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
	query := fmt.Sprintf(`
		SELECT
			id, name
		FROM
			%s
		WHERE
			%s
		ORDER BY
		name`, d.tableName, wherePart)
	rows, err := d.db.Conn.Query(d.db.Ctx, query, params...)
	if err != nil {
		return []*CharacteristicType{}, err
	}
	defer rows.Close()

	var result []*CharacteristicType
	for rows.Next() {
		var entry CharacteristicType
		err := rows.Scan(&entry.Id, &entry.Name)
		if err != nil {
			return []*CharacteristicType{}, err
		}
		result = append(result, &entry)
	}
	if rows.Err() != nil {
		return []*CharacteristicType{}, rows.Err()
	}
	return result, nil
}

func (d *CharacteristicDao) AssignToFace(faceId int64, typeId int64) error {
	_, err := d.db.Conn.Exec(d.db.Ctx, "INSERT INTO "+d.joinTable+"(face_id, type_id) VALUES($1, $2)", faceId, typeId)
	if err != nil {
		return err
	}
	return nil
}

func (d *CharacteristicDao) FindAssignments(faceId int64) ([]*CharacteristicType, error) {
	rows, err := d.db.Conn.Query(d.db.Ctx, `
			SELECT t.id, t.name 
			FROM `+d.tableName+` t JOIN `+d.joinTable+` ct ON t.id = ct.type_id
			WHERE ct.face_id = $1
			ORDER BY t.name`, faceId)
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

func (d *CharacteristicDao) DeleteAssignments(faceId int64, typeIds ...int64) error {
	if len(typeIds) == 0 {
		return nil
	}
	var params []interface{}
	// param $1 is the card id
	params = append(params, faceId)

	var inPart strings.Builder
	for i, id := range typeIds {
		if i > 0 {
			inPart.WriteString(", ")
		}
		params = append(params, id)
		inPart.WriteString(fmt.Sprintf("$%d", len(params)))
	}

	inPart.WriteString(", ")
	inPart.WriteString(fmt.Sprintf("$%d", len(typeIds)+1)) // additional one because of faceId

	query := "DELETE FROM " + d.joinTable + " WHERE face_id = $1 AND type_id IN (" + inPart.String() + ")"

	ct, err := d.db.Conn.Exec(d.db.Ctx, query, params...)
	if err != nil {
		return err
	}
	ra := ct.RowsAffected()
	if ra != int64(len(typeIds)) {
		return fmt.Errorf("expected to deleted %d assigned types but deleted %d from card face with id %d, %s", len(typeIds), ra, faceId, query)
	}
	return nil
}

func (d *CharacteristicDao) DeleteAllAssignments(faceId int64) error {
	query := "DELETE FROM " + d.joinTable + " WHERE face_id = $1"

	_, err := d.db.Conn.Exec(d.db.Ctx, query, faceId)
	if err != nil {
		return err
	}
	return nil
}
