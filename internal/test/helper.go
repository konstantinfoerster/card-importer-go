package test

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func LoadFile(t *testing.T, path string) io.Reader {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err, fmt.Sprintf("failed to open file %s", path))

	return bufio.NewReader(f)
}

func FileContent(t *testing.T, path string) []byte {
	content, err := io.ReadAll(LoadFile(t, path))
	require.NoError(t, err, fmt.Sprintf("failed to read data from %s", path))

	return content
}

//func AssertDeepEqual(t *testing.T, expected interface{}, actual interface{}) {
//	t.Helper()
//
//	assert.IsType(t, expected, actual, "found different types")
//
//	w, err := json.Marshal(expected)
//	require.NoError(t, err, "failed to marshal 'expected' struct")
//
//	g, err := json.Marshal(actual)
//	require.NoError(t, err, "failed to marshal 'actual' struct")
//
//	o := jsondiff.DefaultConsoleOptions()
//
//	d, s := jsondiff.Compare(w, g, &o)
//
//	if d != jsondiff.FullMatch {
//		t.Errorf("found difference in struct, check result below:\n %v", s)
//	}
//}

func NewTmpDirWithCleanup(t *testing.T) string {
	dir, err := os.MkdirTemp("", "downloads")
	require.NoError(t, err, "failed to create temp dir")

	t.Cleanup(cleanup(t, dir))
	return dir
}

func cleanup(t *testing.T, path string) func() {
	return func() {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("failed to delete tmp dir %v", err)
		}
	}
}
