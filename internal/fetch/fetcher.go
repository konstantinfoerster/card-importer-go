package fetch

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Fetcher interface {
	Fetch(url string) (*Response, error)
}

type Response struct {
	ContentType string
	Body        io.ReadCloser
}

func (f *Response) BuildFilename(prefix string) (string, error) {
	if prefix == "" {
		return "", fmt.Errorf("prefix is required to build a filenname")
	}
	contentType := strings.Split(f.ContentType, ";")[0]
	switch contentType {
	case "application/json":
		return prefix + ".json", nil
	case "application/zip":
		return prefix + ".zip", nil
	case "image/jpeg":
		return prefix + ".jpg", nil
	case "image/png":
		return prefix + ".png", nil
	default:
		return "", fmt.Errorf("unsupported content type %s", contentType)
	}
}

var doOnce sync.Once
var client *http.Client

// TODO Maybe use an interface to do the validation e.g. apply(resp *Response) bool
func NewFetcher(allowedTypes []string) Fetcher {
	return &fetcher{
		allowedTypes: allowedTypes,
		timeout:      time.Second * 30,
	}
}

func NewDefaultFetcher() Fetcher {
	allowedTypes := []string{"application/json", "image/jpeg", "image/png"}
	return NewFetcher(allowedTypes)
}

type fetcher struct {
	allowedTypes []string
	timeout      time.Duration
}

func (f *fetcher) getClient() *http.Client {
	doOnce.Do(func() {
		client = &http.Client{
			Timeout: f.timeout,
		}
	})
	return client
}

func (f *fetcher) Fetch(url string) (*Response, error) {
	resp, err := f.getClient().Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
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
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if err != nil {
			return nil, err
		}
		return nil, ExternalApiError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	contentType := resp.Header.Get("content-type")
	if !f.isAllowedContentType(contentType) {
		return nil, fmt.Errorf("unsupported content-type %s", contentType)
	}

	return &Response{
		ContentType: contentType,
		Body:        resp.Body,
	}, nil
}

func (f *fetcher) isAllowedContentType(ct string) bool {
	for _, allowed := range f.allowedTypes {
		t := strings.Split(ct, ";")[0]
		if allowed == t {
			return true
		}
	}
	return false
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
