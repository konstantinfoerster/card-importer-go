package storage

import "io"

type Storage interface {
	Store(in io.Reader, path ...string) (*StoredFile, error)
	Load(path ...string) (io.ReadCloser, error)
}

type StoredFile struct {
	Path         string
	AbsolutePath string
}
