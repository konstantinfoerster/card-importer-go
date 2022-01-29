package api

import "io"

type Report struct {
	CardCount int
	SetCount  int
}

type Importer interface {
	Import(r io.Reader) (*Report, error)
}

type Changeset struct {
	changes map[string]Changes
}

func NewChangeset() Changeset {
	return Changeset{
		changes: map[string]Changes{},
	}
}

type Changes struct {
	From interface{}
	To   interface{}
}

func (c Changeset) Add(field string, changed Changes) {
	c.changes[field] = changed
}

func (c *Changeset) HasChanges() bool {
	return len(c.changes) > 0
}

type Translation interface {
	LangCode() string
	Equal(other Translation) bool
}
