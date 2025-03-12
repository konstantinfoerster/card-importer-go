package mtgjson

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func unzip(src string, dest string) ([]string, error) {
	var readByteLimit int64 = 512 * 1024 * 1024 // 512 MiB
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
				err = errors.Wrap(err, cErr.Error())
			}
		}
	}(r)

	if err = os.MkdirAll(dest, 0750); err != nil {
		return nil, err
	}

	var oneKiB int64 = 1024
	var readBytes int64
	for _, f := range r.File {
		unsafeZipUncompressedSize := f.UncompressedSize64
		if unsafeZipUncompressedSize <= 0 || unsafeZipUncompressedSize > math.MaxInt64 {
			return nil, fmt.Errorf("cannot write file, unrcompressed size is > maxInt64 or <= 0")
		}
		// #nosec G115 false positiv. This bug is fixed in latest gosec but not in golangci
		zipUncompressedSize := int64(unsafeZipUncompressedSize)
		if zipUncompressedSize > readByteLimit {
			return nil, fmt.Errorf("cannot write next file, reached limit of %dMiB", readByteLimit/oneKiB/oneKiB)
		}

		path, err := sanitizeArchivePath(dest, f.Name)
		if err != nil {
			return nil, err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
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

	if err := os.MkdirAll(filepath.Dir(filepath.Dir(destFile)), zippedFile.Mode()); err != nil {
		return "", err
	}

	f, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zippedFile.Mode())
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
				err = errors.Wrap(err, cErr.Error())
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
				err = errors.Wrap(err, cErr.Error())
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

func sanitizeArchivePath(dest, filename string) (string, error) {
	path := filepath.Join(dest, filename)
	if strings.HasPrefix(path, filepath.Clean(dest)) {
		return path, nil
	}

	// Zip slip
	return "", fmt.Errorf("illegal file path %s", path)
}
