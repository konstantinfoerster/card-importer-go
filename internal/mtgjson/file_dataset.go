package mtgjson

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/pkg/errors"
	"io"
	"os"
)

type fileDataset struct {
	dataset   dataset.Dataset
	readLimit int64
}

func NewFileDataset(dataset dataset.Dataset) dataset.Dataset {
	return &fileDataset{
		dataset:   dataset,
		readLimit: 255,
	}
}

func (imp *fileDataset) Import(r io.Reader) (*dataset.Report, error) {
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

	report, err := imp.dataset.Import(f)
	return report, err
}
