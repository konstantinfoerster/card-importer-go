package api

import (
	"io"
)

var SupportedLanguages = []string{"deu", "eng"}

type DatasetReport struct {
	CardCount int
	SetCount  int
}

type Dataset interface {
	Import(r io.Reader) (*DatasetReport, error)
}
