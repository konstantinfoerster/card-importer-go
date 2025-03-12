package cards

import (
	"database/sql"
	"fmt"
	"time"
)

type CardSet struct {
	Code         string
	Name         string
	TotalCount   int
	Released     time.Time // can be null ??
	Block        CardBlock
	Type         string
	Translations []SetTranslation
}

func (s *CardSet) isValid() error {
	if s.Code == "" {
		return fmt.Errorf("field 'code' must not be empty")
	}

	if s.Type == "" {
		return fmt.Errorf("field 'type' must not be empty in set %s", s.Code)
	}

	return nil
}

func (s *CardSet) Diff(other *CardSet) *Changeset {
	changes := NewDiff()

	if other.Block.ID.Valid && other.Block.notEquals(s.Block) {
		changes.Add("Block", Changes{
			From: s.Block,
			To:   other.Block,
		})
	}
	if s.Name != other.Name {
		changes.Add("Name", Changes{
			From: s.Name,
			To:   other.Name,
		})
	}
	if s.Type != other.Type {
		changes.Add("Type", Changes{
			From: s.Type,
			To:   other.Type,
		})
	}
	if !s.Released.Equal(other.Released) {
		changes.Add("Released", Changes{
			From: s.Released,
			To:   other.Released,
		})
	}
	if s.TotalCount != other.TotalCount {
		changes.Add("TotalCount", Changes{
			From: s.TotalCount,
			To:   other.TotalCount,
		})
	}

	return changes
}

type CardBlock struct {
	ID    sql.NullInt64
	Block string
}

func (b CardBlock) notEquals(other CardBlock) bool {
	return b.ID.Int64 != other.ID.Int64
}

type SetTranslation struct {
	Name string
	Lang string
}
