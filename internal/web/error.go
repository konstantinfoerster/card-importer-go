package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

func NewErr(url string, code int, msg string) error {
	return &ExternalAPIError{URL: url, StatusCode: code, Message: msg}
}

func NewHTTPErr(url string, resp *http.Response) error {
	var maxErrorMsgLengthBytes int64 = 2048
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorMsgLengthBytes))
	if err != nil {
		msg := fmt.Errorf("failed to read response body due to %w", err)

		return &ExternalAPIError{URL: url, StatusCode: resp.StatusCode, Message: msg.Error()}
	}

	return &ExternalAPIError{URL: url, StatusCode: resp.StatusCode, Message: string(body)}
}

type ExternalAPIError struct {
	URL        string
	Message    string
	StatusCode int
}

func (e *ExternalAPIError) Error() string {
	return fmt.Sprintf("%d: %s (URL: %s)", e.StatusCode, strings.TrimSpace(e.Message), e.URL)
}

func (e *ExternalAPIError) Is(target error) bool {
	t, ok := target.(*ExternalAPIError)
	if !ok {
		return false
	}

	return e.StatusCode == t.StatusCode
}

func IsStatusCode(err error, statusCode ...int) bool {
	if len(statusCode) == 0 {
		return false
	}

	var apiErr *ExternalAPIError
	if errors.As(err, &apiErr) {
		return slices.Contains(statusCode, apiErr.StatusCode)
	}

	return false
}
