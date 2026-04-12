package mtgjson

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var ErrZipFile = errors.New("invalid zip file")

func unzip(src string, dest string) ([]string, error) {
	var readByteLimit int64 = 1024 * 1024 * 1024 // 1024 MiB
	var files []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer func(toClose *zip.ReadCloser) {
		cErr := toClose.Close()
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Join(err, cErr)
			}
		}
	}(r)

	// #nosec G703 is already sanitized
	if err = os.MkdirAll(dest, 0700); err != nil {
		return nil, err
	}

	var oneKiB int64 = 1024
	var readBytes int64
	for _, f := range r.File {
		unsafeZipUncompressedSize := f.UncompressedSize64
		if unsafeZipUncompressedSize <= 0 || unsafeZipUncompressedSize > math.MaxInt64 {
			return nil, fmt.Errorf("cannot write file, unrcompressed size is > maxInt64 or <= 0")
		}
		zipUncompressedSize := int64(unsafeZipUncompressedSize)
		if zipUncompressedSize > readByteLimit {
			return nil, fmt.Errorf("cannot write next file, reached limit of %dMiB", readByteLimit/oneKiB/oneKiB)
		}

		path, err := SanitizePath(dest, f.Name)
		if err != nil {
			return nil, err
		}

		if f.FileInfo().IsDir() {
			// #nosec G703 is already sanitized
			if err := os.MkdirAll(path, 0700); err != nil {
				return nil, err
			}

			continue
		}

		// prevent zip bombs
		readBytes += zipUncompressedSize
		if readBytes > readByteLimit {
			return nil, fmt.Errorf("cannot write next file, reached limit of %dMiB", readByteLimit/oneKiB/oneKiB)
		}

		d, err := writeFile(f, path, zipUncompressedSize)
		if err != nil {
			return nil, err
		}
		files = append(files, d)
	}

	return files, err
}

func writeFile(zippedFile *zip.File, destFile string, readBytesN int64) (string, error) {
	destFile = filepath.Clean(destFile)

	// #nosec G703 is already sanitized
	if err := os.MkdirAll(filepath.Dir(destFile), 0700); err != nil {
		return "", err
	}

	// #nosec G703 is already sanitized
	f, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer func(toClose *os.File) {
		cErr := toClose.Close()
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Join(err, cErr)
			}
		}
	}(f)

	rc, err := zippedFile.Open()
	if err != nil {
		return "", err
	}
	defer func(toClose io.ReadCloser) {
		cErr := toClose.Close()
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Join(err, cErr)
			}
		}
	}(rc)

	if _, err := io.CopyN(f, rc, readBytesN); err != nil {
		return "", err
	}

	if err := f.Sync(); err != nil {
		return "", err
	}

	return destFile, err
}

func SanitizePath(dest, filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename should not be empty, %w", ErrZipFile)
	}

	if filepath.IsAbs(filename) {
		return "", fmt.Errorf("filename should not be absolute %s, %w", filename, ErrZipFile)
	}

	if !filepath.IsAbs(dest) {
		return "", fmt.Errorf("dest path must be absolute %s, %w", dest, ErrZipFile)
	}

	dest = filepath.Clean(dest)
	path := filepath.Join(dest, filename)
	// path should start with e.g. /out/ so we do not allow e.g /out-tmp
	if !strings.HasPrefix(path, dest+string(filepath.Separator)) {
		// Zip slip
		return "", fmt.Errorf("illegal file path %s, %w", path, ErrZipFile)
	}

	return path, nil
}
