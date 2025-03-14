package web

import (
	"fmt"
	"strings"
)

const (
	MimeTypeJSON     = "application/json"
	MimeTypeJpeg     = "image/jpeg"
	MimeTypePng      = "image/png"
	MimeTypeZip      = "application/zip"
	HeaderAccept     = "Accept"
	HeaderUserAgent  = "User-Agent"
	DefaultUserAgent = "CardImporter/0.1"
)

// NewMimeType creates a MimeType from the given content-type.
func NewMimeType(contentType string) MimeType {
	ct := strings.Split(contentType, ";")[0]

	return MimeType{value: strings.TrimSpace(strings.ToLower(ct))}
}

type MimeType struct {
	value string
}

// BuildFilename appends a file extension to the given name.
// An error is returned for unsupported mime-types or empty names.
func (m MimeType) BuildFilename(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("can't build file name without prefix")
	}

	switch m.value {
	case MimeTypeJSON:
		return name + ".json", nil
	case MimeTypeZip:
		return name + ".zip", nil
	case MimeTypeJpeg:
		return name + ".jpg", nil
	case MimeTypePng:
		return name + ".png", nil
	default:
		return "", fmt.Errorf("unsupported mime type %s", m.value)
	}
}

// IsZip returns true if mime-type is application/zip.
func (m MimeType) IsZip() bool {
	return m.value == MimeTypeZip
}

// Raw returns the extracted mime-type.
func (m MimeType) Raw() string {
	return m.value
}
