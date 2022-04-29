package storage

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func NewLocalStorage(config config.Storage) (Storage, error) {
	err := os.MkdirAll(config.Location, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage dir %s %w", config.Location, err)
	}
	return &localStorage{
		config: config,
	}, nil
}

type localStorage struct {
	config config.Storage
}

func (s *localStorage) fromBasePath(path ...string) string {
	var escapeSafe []string
	for _, p := range path {
		if strings.TrimSpace(p) == ".." {
			continue
		}
		escapeSafe = append(escapeSafe, p)
	}

	file := filepath.Join(s.config.Location, filepath.Join(escapeSafe...))
	file = filepath.Clean(file)

	return file
}

func (s *localStorage) Store(r io.Reader, path ...string) (*StoredFile, error) {
	filePath := s.fromBasePath(path...)
	if len(path) > 1 {
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create sub dirs for %s %w", filePath, err)
		}
	}

	flags := os.O_RDWR | os.O_CREATE
	if s.config.Mode == config.REPLACE {
		flags |= os.O_TRUNC // truncate existing file
	} else {
		flags |= os.O_EXCL // file must not exist
	}

	target, err := os.OpenFile(filePath, flags, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create empty file %s with mode %s %w", filePath, s.config.Mode, err)
	}
	defer func(toClose *os.File) {
		cErr := toClose.Close()
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Wrap(err, cErr.Error())
			}
		}
	}(target)

	_, err = io.Copy(target, r)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file %w", err)
	}
	err = target.Sync()
	if err != nil {
		return nil, fmt.Errorf("failed to sync file %w", err)
	}

	return &StoredFile{
		AbsolutePath: filePath,
		Path:         s.removeBasePath(filePath),
	}, err
}

func (s *localStorage) removeBasePath(path string) string {
	noBasePath := strings.TrimPrefix(path, s.config.Location)
	noBasePath = strings.TrimPrefix(noBasePath, "/")
	return noBasePath
}

func (s *localStorage) Load(path ...string) (io.ReadCloser, error) {
	filePath := s.fromBasePath(path...)

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info %s %w", filePath, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("loading a directory is not supported")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s %w", filePath, err)
	}

	return file, nil
}
