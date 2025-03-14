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

	// #nosec G304 only used in tests
	f, err := os.Open(path)
	require.NoError(t, err, fmt.Sprintf("failed to open file %s", path))

	return bufio.NewReader(f)
}
