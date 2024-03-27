package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

const (
	MimeTypeJSON = "application/json"
	MimeTypeJpeg = "image/jpeg"
	MimeTypePng  = "image/png"
	MimeTypeZip  = "application/zip"
)

func NewMimeType(mimeType string) MimeType {
	return MimeType{Value: strings.TrimSpace(strings.ToLower(mimeType))}
}

type MimeType struct {
	Value string
}

func (m MimeType) BuildFilename(prefix string) (string, error) {
	if strings.TrimSpace(prefix) == "" {
		return "", fmt.Errorf("can't build file name without prefix")
	}
	switch m.Value {
	case MimeTypeJSON:
		return prefix + ".json", nil
	case MimeTypeZip:
		return prefix + ".zip", nil
	case MimeTypeJpeg:
		return prefix + ".jpg", nil
	case MimeTypePng:
		return prefix + ".png", nil
	default:
		return "", fmt.Errorf("unsupported mime type %s", m.Value)
	}
}

func (m MimeType) IsZip() bool {
	return m.Value == MimeTypeZip
}

func (m MimeType) Raw() string {
	return m.Value
}

type Response struct {
	Body        io.Reader
	ContentType string
}

func (r *Response) MimeType() MimeType {
	return NewMimeType(strings.Split(r.ContentType, ";")[0])
}

type Fetcher interface {
	Fetch(url string, handleResponse func(resp *Response) error) error
}

func NewFetcher(client *http.Client, validator ...Validator) Fetcher {
	return &fetcher{
		validators: validator,
		client:     client,
	}
}

type fetcher struct {
	validators []Validator
	client     *http.Client
}

func (f *fetcher) Fetch(url string, handleResponse func(resp *Response) error) error {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := f.client.Do(req) //nolint:bodyclose
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
		var maxErrorMsgLengthBytes int64 = 2048
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorMsgLengthBytes))
		if err != nil {
			return fmt.Errorf("failed to read response body for %s %w", url, err)
		}

		return ExternalAPIError{StatusCode: resp.StatusCode, Message: string(body)}
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

type ExternalAPIError struct {
	Message    string
	StatusCode int
}

func (e ExternalAPIError) Error() string {
	return fmt.Sprintf("%d:%s", e.StatusCode, e.Message)
}

func (e ExternalAPIError) Is(target error) bool {
	var apiErr *ExternalAPIError
	if errors.As(target, &apiErr) {
		return apiErr.StatusCode == e.StatusCode
	}

	return false
}

var ErrNotFound = &ExternalAPIError{StatusCode: 404, Message: "Not found"}
