package fetch

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	MimeTypeJson = "application/json"
	MimeTypeJpeg = "image/jpeg"
	MimeTypePng  = "image/png"
	MimeTypeZip  = "application/zip"
)

func NewMimeType(mimeType string) MimeType {
	return MimeType{string: strings.TrimSpace(strings.ToLower(mimeType))}
}

type MimeType struct {
	string
}

func (m MimeType) BuildFilename(prefix string) (string, error) {
	if strings.TrimSpace(prefix) == "" {
		return "", fmt.Errorf("can't build file name without prefix")
	}
	switch m.string {
	case MimeTypeJson:
		return prefix + ".json", nil
	case MimeTypeZip:
		return prefix + ".zip", nil
	case MimeTypeJpeg:
		return prefix + ".jpg", nil
	case MimeTypePng:
		return prefix + ".png", nil
	default:
		return "", fmt.Errorf("unsupported mime type %s", m.string)
	}
}

func (m *MimeType) IsZip() bool {
	return m.string == MimeTypeZip
}

func (m *MimeType) Raw() string {
	return m.string
}

type Response struct {
	ContentType string
	Body        io.Reader
}

func (r *Response) MimeType() MimeType {
	return NewMimeType(strings.Split(r.ContentType, ";")[0])
}

type Fetcher interface {
	Fetch(url string, handleResponse func(resp *Response) error) error
}

var doOnce sync.Once
var client *http.Client
var DefaultAllowedTypes = []string{MimeTypeJson, MimeTypeJpeg, MimeTypePng}

func NewFetcher(cfg config.Http, validator ...Validator) Fetcher {
	return &fetcher{
		validators: validator,
		timeout:    cfg.Timeout,
	}
}

type fetcher struct {
	validators []Validator
	timeout    time.Duration
}

func (f *fetcher) getClient() *http.Client {
	doOnce.Do(func() {
		client = &http.Client{
			Timeout: f.timeout,
		}
	})
	return client
}

func (f *fetcher) Fetch(url string, handleResponse func(resp *Response) error) error {
	resp, err := f.getClient().Get(url)
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
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2048)) // limited size for the error message
		if err != nil {
			return err
		}
		return ExternalApiError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	for _, v := range f.validators {
		if err := v.Apply(resp); err != nil {
			return err
		}
	}

	return handleResponse(&Response{
		ContentType: resp.Header.Get("content-type"),
		Body:        resp.Body,
	})
}

type ExternalApiError struct {
	StatusCode int
	Message    string
}

func (e ExternalApiError) Error() string {
	return fmt.Sprintf("%d:%s", e.StatusCode, e.Message)
}

func (e ExternalApiError) Is(target error) bool {
	t, ok := target.(*ExternalApiError)
	if !ok {
		return false
	}
	return t.StatusCode == e.StatusCode
}

var NotFoundError = &ExternalApiError{StatusCode: 404, Message: "Not found"}
