package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/pkg/errors"
)

type Storer interface {
	Store(in io.Reader, path ...string) (StoredFile, error)
	Load(path ...string) (io.ReadCloser, error)
}

type StoredFile struct {
	Path         string
	AbsolutePath string
}

func NewLocalStorage(config config.Storage) (Storer, error) {
	if err := os.MkdirAll(config.Location, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage dir %s %w", config.Location, err)
	}

	return &localStorage{
		config: config,
	}, nil
}

type localStorage struct {
	config config.Storage
}

func (s *localStorage) fromBasePath(path ...string) (string, error) {
	baseDir := s.config.Location
	targetDir := filepath.Join(baseDir, filepath.Join(path...))
	targetDir = filepath.Clean(targetDir)

	if !strings.HasPrefix(targetDir, baseDir) {
		return "", fmt.Errorf("path is not within base path, %s", baseDir)
	}

	return targetDir, nil
}

func (s *localStorage) Store(r io.Reader, path ...string) (StoredFile, error) {
	filePath, err := s.fromBasePath(path...)
	if err != nil {
		return StoredFile{}, err
	}

	if len(path) > 1 {
		if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
			return StoredFile{}, fmt.Errorf("failed to create sub dirs for %s %w", filePath, err)
		}
	}

	flags := os.O_RDWR | os.O_CREATE
	if s.config.Mode == config.REPLACE {
		flags |= os.O_TRUNC // truncate existing file
	} else {
		flags |= os.O_EXCL // file must not exist
	}

	// #nosec G304 fromBasePath does already a path cleanup
	target, err := os.OpenFile(filePath, flags, 0600)
	if err != nil {
		return StoredFile{}, fmt.Errorf("failed to create empty file %s with mode %s %w", filePath, s.config.Mode, err)
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

	if _, err = io.Copy(target, r); err != nil {
		return StoredFile{}, fmt.Errorf("failed to copy file %w", err)
	}

	if err = target.Sync(); err != nil {
		return StoredFile{}, fmt.Errorf("failed to sync file %w", err)
	}

	return StoredFile{
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
	filePath, err := s.fromBasePath(path...)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info %s %w", filePath, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("loading a directory is not supported")
	}

	// #nosec G304 fromBasePath does already a path cleanup
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s %w", filePath, err)
	}

	return file, nil
}
