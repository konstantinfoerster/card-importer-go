package mtgjson

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/pkg/errors"
	"io"
	"os"
)

type FileImport struct {
	importer  api.Importer
	readLimit int64
}

func NewFileImport(importer api.Importer) *FileImport {
	return &FileImport{
		importer:  importer,
		readLimit: 255,
	}
}

func (imp *FileImport) Import(r io.Reader) (*api.Report, error) {
	rLimit := &io.LimitedReader{
		R: r,
		N: imp.readLimit + 1, // + 1 to check if we read more bytes than expected
	}
	filePath, err := io.ReadAll(rLimit)
	if err != nil {
		return nil, err
	}
	if rLimit.N == 0 {
		return nil, fmt.Errorf("file path must be <= %d characters", imp.readLimit)
	}

	f, err := os.Open(string(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s %w", filePath, err)
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

	report, err := imp.importer.Import(f)
	return report, err
}
