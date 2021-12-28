package api

import "io"

type Report struct {
	CardCount int
	SetCount  int
}

type Importer interface {
	Import(r io.Reader) (*Report, error)
}
