package diff

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
