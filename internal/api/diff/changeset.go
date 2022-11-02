package diff

import (
	"fmt"
	"strings"
)

type Changeset struct {
	changes map[string]Changes
}

func New() *Changeset {
	return &Changeset{
		changes: map[string]Changes{},
	}
}

type Changes struct {
	From interface{}
	To   interface{}
}

func (c *Changeset) Add(field string, changed Changes) {
	c.changes[field] = changed
}

func (c *Changeset) HasChanges() bool {
	return len(c.changes) > 0
}

func (c *Changeset) String() string {
	var changes []string
	for k, v := range c.changes {
		changes = append(changes, fmt.Sprintf("Field '%s' from '%v' to '%v'", k, v.From, v.To))
	}
	return strings.Join(changes, ", ")
}
