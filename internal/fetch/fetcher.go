package fetch

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Response struct {
	ContentType string
	Body        io.Reader
}

type Fetcher interface {
	Fetch(url string) (*Response, error)
}

var doOnce sync.Once
var client *http.Client
var DefaultBodyLimit int64 = 2 * 1024 // in kb
var DefaultAllowedTypes = []string{"application/json", "image/jpeg", "image/png"}

// TODO Maybe use an interface to do the validation e.g. apply(resp *Response) bool
func NewFetcher(allowedTypes []string, bodyLimit int64) Fetcher {
	return &fetcher{
		allowedTypes: allowedTypes,
		timeout:      time.Second * 30,
		bodyLimit:    bodyLimit,
	}
}

type fetcher struct {
	allowedTypes []string
	timeout      time.Duration
	bodyLimit    int64 // in kb
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
			return nil, err
		}
		return nil, ExternalApiError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	contentType := resp.Header.Get("content-type")
	if !f.isAllowedContentType(contentType) {
		return nil, fmt.Errorf("unsupported content-type %s", contentType)
	}

	content, err := f.copyWithLimit(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		ContentType: contentType,
		Body:        bytes.NewBuffer(content),
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

func (f *fetcher) copyWithLimit(r io.Reader) ([]byte, error) {
	rLimit := &io.LimitedReader{
		R: r,
		N: (f.bodyLimit * 1024) + 1, // + 1 to check if we read more bytes than expected
	}
	content, err := io.ReadAll(rLimit)
	if err != nil {
		return nil, err
	}
	if rLimit.N == 0 {
		return nil, fmt.Errorf("body must be <= %d kilobytes", f.bodyLimit)
	}

	return content, nil
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
