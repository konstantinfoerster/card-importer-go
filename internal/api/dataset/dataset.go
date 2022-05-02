package dataset

import (
	"io"
)

var SupportedLanguages = []string{"deu", "eng"}

type Report struct {
	CardCount int
	SetCount  int
}

type Dataset interface {
	Import(r io.Reader) (*Report, error)
}
