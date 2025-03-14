package cards

import (
	"io"
)

type Report struct {
	CardCount int
	SetCount  int
}

type Dataset interface {
	Import(r io.Reader) (*Report, error)
}
