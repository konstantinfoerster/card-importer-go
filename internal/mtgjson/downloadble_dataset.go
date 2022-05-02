package mtgjson

import (
	"archive/zip"
	"fmt"
	dataset2 "github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type downloadableDataset struct {
	dataset   dataset2.Dataset
	readLimit int64
}

func NewDownloadableDataset(dataset dataset2.Dataset) dataset2.Dataset {
	return &downloadableDataset{
		dataset:   dataset,
		readLimit: 100, // max length of the url
	}
}

type downloadedFile struct {
	contentType string
	filepath    string
}

func (f *downloadedFile) isZip() bool {
	return f.contentType == "application/zip"
}

func (imp *downloadableDataset) Import(r io.Reader) (*dataset2.Report, error) {
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

	dFile, err := download(string(url))
	if err != nil {
		return nil, err
	}

	var fileToImport string
	if dFile.isZip() {
		dest := filepath.Dir(dFile.filepath)
		files, err := unzip(dFile.filepath, dest)
		if err != nil {
			return nil, err
		}
		if len(files) != 1 {
			return nil, fmt.Errorf("unexpected file count inside zip file, expected 1 but found %d", len(files))
		}
		fileToImport = files[0]
	} else {
		fileToImport = dFile.filepath
	}

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

var doOnce sync.Once
var client *http.Client

func download(url string) (*downloadedFile, error) {
	log.Info().Msgf("Downloading %s", url)
	doOnce.Do(func() {
		client = &http.Client{
			Timeout: time.Second * 100,
		}
	})

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
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
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to download file (%d) reason: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("content-type")
	var fileName string
	switch contentType {
	case "application/json":
		fileName = fmt.Sprintf("%d.json", time.Now().UnixMilli())
	case "application/zip":
		fileName = fmt.Sprintf("%d.zip", time.Now().UnixMilli())
	default:
		return nil, fmt.Errorf("unsupported content-type %s", contentType)
	}

	targetFile, err := createTmpTargetFile(fileName)
	if err != nil {
		return nil, err
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
	}(targetFile)

	_, err = io.Copy(targetFile, resp.Body)
	if err != nil {
		return nil, err
	}
	err = targetFile.Sync()
	if err != nil {
		return nil, err
	}

	filePath, err := filepath.Abs(targetFile.Name())
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("download finished and stored at of %s", filePath)
	return &downloadedFile{
		contentType: contentType,
		filepath:    filePath,
	}, err
}

func unzip(src string, dest string) ([]string, error) {
	log.Info().Msgf("Unzipping %s to %s", src, dest)
	var readByteLimit uint64 = 300 * 1024 * 1024 // 300 MB
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

	if err := os.MkdirAll(dest, 0755); err != nil {
		return nil, err
	}

	var readBytes uint64
	writeFile := func(zippedFile *zip.File, destFile string) error {
		// prevent zip bombs
		readBytes += zippedFile.UncompressedSize64
		if readBytes > readByteLimit {
			return fmt.Errorf("failed to write next file, reached limit of %dMB", readByteLimit/1024/1024)
		}

		if err := os.MkdirAll(filepath.Dir(filepath.Dir(destFile)), zippedFile.Mode()); err != nil {
			return err
		}

		f, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zippedFile.Mode())
		if err != nil {
			return err
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
			return err
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

		if _, err = io.Copy(f, rc); err != nil {
			return err
		}
		if err := f.Sync(); err != nil {
			return err
		}

		files = append(files, destFile)

		return err
	}
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)

		// check for ZipSlip
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path %s", path)
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(path, f.Mode())
			if err != nil {
				return nil, err
			}
			continue
		}

		if err := writeFile(f, path); err != nil {
			return nil, err
		}
	}

	log.Info().Msgf("Unzip finished with files %v", files)
	return files, err
}

func createTmpTargetFile(fileName string) (*os.File, error) {
	tmpDir, err := os.MkdirTemp("", "downloads")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp download dir %w", err)
	}

	targetFile := filepath.Join(tmpDir, fileName)
	out, err := os.Create(targetFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp file %s, %w", targetFile, err)
	}
	return out, nil
}
