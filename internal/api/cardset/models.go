package cardset

import (
	"database/sql"
	"time"
)

type CardSet struct {
	Code         string
	Name         string
	TotalCount   int
	Released     time.Time
	Block        CardBlock
	Type         string
	Translations []Translation
}

type CardBlock struct {
	Id    sql.NullInt64
	Block string
}

func (b CardBlock) notEquals(other CardBlock) bool {
	return b.Id.Int64 != other.Id.Int64
}

type Translation struct {
	Name string
	Lang string
}
