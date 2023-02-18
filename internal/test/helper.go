package test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func LoadFile(t *testing.T, path string) io.Reader {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err, fmt.Sprintf("failed to open file %s", path))

	return bufio.NewReader(f)
}

func FileContent(t *testing.T, path string) []byte {
	t.Helper()

	content, err := io.ReadAll(LoadFile(t, path))
	require.NoError(t, err, fmt.Sprintf("failed to read data from %s", path))

	return content
}

func NewTmpDirWithCleanup(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "downloads")
	require.NoError(t, err, "failed to create temp dir")

	t.Cleanup(Cleanup(t, dir))

	return dir
}

func Cleanup(t *testing.T, path string) func() {
	t.Helper()

	return func() {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("failed to delete tmp dir %v", err)
		}
	}
}
