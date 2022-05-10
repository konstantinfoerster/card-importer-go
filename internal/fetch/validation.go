package fetch

import (
	"fmt"
	"net/http"
	"strings"
)

type Validator interface {
	Apply(resp *http.Response) error
}

func NewContentTypeValidator(allowedTypes []string) Validator {
	return ContentTypeValidator{
		allowedTypes: allowedTypes,
	}
}

type ContentTypeValidator struct {
	allowedTypes []string
}

func (v ContentTypeValidator) Apply(resp *http.Response) error {
	contentType := resp.Header.Get("content-type")
	for _, allowed := range v.allowedTypes {
		t := strings.Split(contentType, ";")[0]
		if allowed == t {
			return nil
		}
	}
	return fmt.Errorf("unsupported content-type %s", contentType)
}
