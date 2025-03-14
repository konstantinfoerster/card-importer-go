package web_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
)

func TestErrorIs(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		targetErr error
		match     bool
	}{
		{
			name:      "is 404 API error",
			err:       web.NewErr("https://localhost", http.StatusNotFound, "not found"),
			targetErr: web.NewErr("https://localhost", http.StatusNotFound, "not found again"),
			match:     true,
		},
		{
			name:      "wrapped is 404 API error",
			err:       fmt.Errorf("not found %w", web.NewErr("https://localhost", http.StatusNotFound, "not found")),
			targetErr: web.NewErr("https://localhost", http.StatusNotFound, "not found again"),
			match:     true,
		},
		{
			name:      "different status code",
			err:       web.NewErr("https://localhost", http.StatusBadRequest, "bad request"),
			targetErr: web.NewErr("https://localhost", http.StatusNotFound, "not found again"),
			match:     false,
		},
		{
			name:      "wrapped different status code",
			err:       fmt.Errorf("some error %w", web.NewErr("https://localhost", http.StatusBadRequest, "bad request")),
			targetErr: web.NewErr("https://localhost", http.StatusNotFound, "not found again"),
			match:     false,
		},
		{
			name:      "no API error",
			err:       web.NewErr("https://localhost", http.StatusBadRequest, "bad request"),
			targetErr: fmt.Errorf("some error"),
			match:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.match, errors.Is(tc.err, tc.targetErr))
		})
	}
}

func TestIsStatusCode(t *testing.T) {
	cases := []struct {
		name        string
		err         error
		statusCodes []int
		match       bool
	}{
		{
			name:        "match error",
			err:         web.NewErr("https://localhost", http.StatusNotFound, ""),
			statusCodes: []int{http.StatusNotFound},
			match:       true,
		},
		{
			name:        "match wrapped error",
			err:         fmt.Errorf("not found %w", web.NewErr("https://localhost", http.StatusNotFound, "")),
			statusCodes: []int{http.StatusNotFound},
			match:       true,
		},
		{
			name:        "match error with multiple status code",
			err:         web.NewErr("https://localhost", http.StatusNotFound, ""),
			statusCodes: []int{400, 429, http.StatusNotFound},
			match:       true,
		},
		{
			name:        "does not match error",
			err:         web.NewErr("https://localhost", http.StatusBadRequest, ""),
			statusCodes: []int{http.StatusNotFound},
			match:       false,
		},
		{
			name:        "does not match wrapped error",
			err:         fmt.Errorf("bad request %w", web.NewErr("https://localhost", http.StatusBadRequest, "")),
			statusCodes: []int{http.StatusNotFound},
			match:       false,
		},
		{
			name:        "does not match different error type",
			err:         fmt.Errorf("some error"),
			statusCodes: []int{http.StatusNotFound},
			match:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.match, web.IsStatusCode(tc.err, tc.statusCodes...))
		})
	}
}
