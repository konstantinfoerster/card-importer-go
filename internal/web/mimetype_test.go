package web_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/web"
	"github.com/stretchr/testify/assert"
)

func TestMimeTypeRaw(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		want        string
	}{
		{
			name:        "from content-type",
			contentType: "application/json",
			want:        "application/json",
		},
		{
			name:        "from content-type with spaced charset",
			contentType: "application/json; charset=utf-8",
			want:        "application/json",
		},
		{
			name:        "from content-type with charset",
			contentType: "application/json;charset=utf-8",
			want:        "application/json",
		},
		{
			name:        "from content-type with charset and boundary",
			contentType: "application/json; charset=utf-8; boundary=A",
			want:        "application/json",
		},
		{
			name:        "from random content-type",
			contentType: "a",
			want:        "a",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := web.NewMimeType(tc.contentType).Raw()

			assert.Equal(t, tc.want, got)
		})
	}
}
