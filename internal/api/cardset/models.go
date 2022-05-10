package cardset

import (
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/diff"
	"time"
)

type CardSet struct {
	Code         string
	Name         string
	TotalCount   int
	Released     time.Time // can be null ??
	Block        CardBlock
	Type         string
	Translations []Translation
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

func (s CardSet) Diff(other *CardSet) *diff.Changeset {
	changes := diff.NewChangeset()

	if other.Block.Id.Valid && other.Block.notEquals(s.Block) {
		changes.Add("Block", diff.Changes{
			From: s.Block,
			To:   other.Block,
		})
	}
	if s.Name != other.Name {
		changes.Add("Name", diff.Changes{
			From: s.Name,
			To:   other.Name,
		})
	}
	if s.Type != other.Type {
		changes.Add("Type", diff.Changes{
			From: s.Type,
			To:   other.Type,
		})
	}
	if !s.Released.Equal(other.Released) {
		changes.Add("Released", diff.Changes{
			From: s.Released,
			To:   other.Released,
		})
	}
	if s.TotalCount != other.TotalCount {
		changes.Add("TotalCount", diff.Changes{
			From: s.TotalCount,
			To:   other.TotalCount,
		})
	}

	return &changes
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
