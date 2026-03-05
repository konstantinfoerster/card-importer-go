package mtgjson_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/mtgjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeArchivePath(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "output")
	err := os.MkdirAll(dest, 0700)
	require.NoError(t, err)

	testcases := []struct {
		name        string
		filename    string
		expected    string
		expectedErr error
	}{
		{
			name:     "normal file",
			filename: "test.txt",
			expected: filepath.Join(dest, "test.txt"),
		},
		{
			name:     "file with sub-dir",
			filename: "sub/test.txt",
			expected: filepath.Join(dest, "sub", "test.txt"),
		},
		{
			name:        "attack 1",
			filename:    "../../../etc/passwd",
			expectedErr: mtgjson.ErrZipFile,
		},
		{
			name:        "attack 2",
			filename:    "subdir/../../../../etc/shadow",
			expectedErr: mtgjson.ErrZipFile,
		},
		{
			name:        "attack 3",
			filename:    "../output-secret/test.txt",
			expectedErr: mtgjson.ErrZipFile,
		},
		{
			name:        "attack 4",
			filename:    "/tmp/malicious.sh",
			expectedErr: mtgjson.ErrZipFile,
		},
		{
			name:        "attack 5",
			filename:    "",
			expectedErr: mtgjson.ErrZipFile,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := mtgjson.SanitizePath(dest, tc.filename)

			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
			}
			assert.Equal(t, tc.expected, actual)
		})
	}
}
