package mtgjson

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type downloadableDataset struct {
	dataset   dataset.Dataset
	fetcher   fetch.Fetcher
	store     storage.Storage
	readLimit int64
}

func NewDownloadableDataset(dataset dataset.Dataset, fetcher fetch.Fetcher, store storage.Storage) dataset.Dataset {
	var maxURLLengthBytes int64 = 100

	return &downloadableDataset{
		dataset:   dataset,
		fetcher:   fetcher,
		store:     store,
		readLimit: maxURLLengthBytes,
	}
}

type downloadedFile struct {
	mimeType fetch.MimeType
	filepath string
}

// Get Returns the file path to the downloaded file. If the file is a zip, it will be extracted and
// expected to contain exactly one file. In that case the path to the extracted file will be returned.
func (d downloadedFile) Get() (string, error) {
	if !d.mimeType.IsZip() {
		return d.filepath, nil
	}
	var err error
	defer func(name string) {
		rErr := os.Remove(name)
		if rErr != nil {
			// report remove errors
			if err == nil {
				err = rErr
			} else {
				err = errors.Wrap(err, rErr.Error())
			}
		} else {
			log.Info().Msgf("Delete zip file %s", name)
		}
	}(d.filepath)

	dest := filepath.Dir(d.filepath)
	files, err := unzip(d.filepath, dest)
	if err != nil {
		return "", err
	}

	if len(files) != 1 {
		return "", fmt.Errorf("unexpected file count inside zip file, expected 1 but found %d", len(files))
	}

	return files[0], err
}

func (imp *downloadableDataset) Import(r io.Reader) (*dataset.Report, error) {
	rLimit := &io.LimitedReader{
		R: r,
		N: imp.readLimit + 1, // + 1 to check if we read more bytes than expected
	}
	url, err := io.ReadAll(rLimit)
	if err != nil {
		return nil, err
	}
	if rLimit.N == 0 {
		return nil, fmt.Errorf("url must be <= %d characters", imp.readLimit)
	}

	var dFile *downloadedFile
	afterDownload := func(resp *fetch.Response) error {
		filename, err := resp.MimeType().BuildFilename(fmt.Sprintf("%d", time.Now().UnixMilli()))
		if err != nil {
			return err
		}
		sFile, err := imp.store.Store(resp.Body, "downloads", filename)
		if err != nil {
			return err
		}
		dFile = &downloadedFile{
			mimeType: resp.MimeType(),
			filepath: sFile.AbsolutePath,
		}

		return nil
	}
	log.Info().Msgf("Downloading %s", url)
	err = imp.fetcher.Fetch(string(url), afterDownload)
	if err != nil {
		return nil, err
	}

	fileToImport, err := dFile.Get()
	if err != nil {
		return nil, err
	}

	fileToImport = filepath.Clean(fileToImport)
	f, err := os.Open(fileToImport)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s %w", fileToImport, err)
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

	report, err := imp.dataset.Import(f)

	return report, err
}

func unzip(src string, dest string) ([]string, error) {
	var readByteLimit int64 = 512 * 1024 * 1024 // 512 MiB
	log.Info().Msgf("Unzipping %s to %s with a target limit of %d bytes", src, dest, readByteLimit)
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

	if err := os.MkdirAll(dest, 0750); err != nil {
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

	log.Info().Msgf("Unzip finished with files %v", files)

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
