package storage_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/konstantinfoerster/card-importer-go/internal/storage"
	"github.com/konstantinfoerster/card-importer-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logger.SetupConsoleLogger()
	err := logger.SetLogLevel("warn")
	if err != nil {
		fmt.Printf("Failed to set log level %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestStoreFailsIfFileIsOutsideBasePath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	path := []string{"..", "dir", "..", "test.txt"}

	_, err = store.Store(strings.NewReader("content"), path...)
	require.Error(t, err, "expected store to failed for path outside base dir")
}

func TestStoreWithSubDirs(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	path := []string{"dir", "sub", "sub2", "sub3", "test.txt"}

	f, err := store.Store(strings.NewReader("content"), path...)
	require.NoError(t, err, "failed to store file")

	assert.FileExists(t, f.AbsolutePath)
}

func TestStoreReturnsCorrectPath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	path := []string{"dir", "test.txt"}

	f, err := store.Store(strings.NewReader("content"), path...)
	assert.NoError(t, err, "failed to store file")

	assert.Equal(t, filepath.Join(dir, "dir", "test.txt"), f.AbsolutePath)
	assert.Equal(t, filepath.Join("dir", "test.txt"), f.Path)
}

func TestStoreModeCreate(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	fileName := "test.txt"

	f, err := store.Store(strings.NewReader("content"), fileName)
	assert.NoError(t, err, "failed to store file")

	assert.FileExists(t, f.AbsolutePath)
	assertFileContent(t, "content", f.AbsolutePath)
}

func TestStoreModeCreateFails(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	fileName := "test.txt"

	_, err = store.Store(strings.NewReader("content"), fileName)
	assert.NoError(t, err, "failed to store file")

	_, err = store.Store(strings.NewReader("differentContent"), fileName)

	assert.Error(t, err, "expected store to fail if file already exists")
	assert.ErrorIs(t, err, os.ErrExist)
}

func TestStoreModeReplace(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.REPLACE,
	})
	require.NoError(t, err)
	fileName := "test.txt"

	_, err = store.Store(strings.NewReader("content"), fileName)
	assert.NoError(t, err, "failed to store file")

	f, err := store.Store(strings.NewReader("differentContent"), fileName)
	assert.NoError(t, err, "failed to store file")

	assert.FileExists(t, f.AbsolutePath)
	assertFileContent(t, "differentContent", f.AbsolutePath)
}

func TestLoadNoneExistingFile(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
	})
	require.NoError(t, err)
	fileName := "notFound.txt"

	_, err = store.Load(fileName)

	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadWithoutAnyPath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
	})
	require.NoError(t, err)

	_, err = store.Load("")
	assert.Error(t, err, "expected not found error but got no error")

	assert.Contains(t, err.Error(), "not supported")
}

func TestLoadOutSideBasePath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	fileName := "test.txt"
	_, err = store.Store(strings.NewReader("doNotCare"), fileName)
	require.NoErrorf(t, err, "failed to store file %s at %s", fileName, dir)

	_, err = store.Load("..", "..", fileName)
	assert.Errorf(t, err, "load outside base path should fail")
}

func TestLoad(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := storage.NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	require.NoError(t, err)
	fileName := "test.txt"
	_, err = store.Store(strings.NewReader("content"), fileName)
	require.NoErrorf(t, err, "failed to store file %s at %s", fileName, dir)

	actual, err := store.Load(fileName)
	require.NoError(t, err)
	defer actual.Close()

	assertContentEquals(t, "content", actual)
}

func assertContentEquals(t *testing.T, expected string, r io.Reader) {
	t.Helper()

	actual, err := io.ReadAll(r)
	require.NoError(t, err, "failed to read file")

	assert.Equal(t, expected, string(actual))
}

func assertFileContent(t *testing.T, expected string, path string) {
	t.Helper()

	actual, err := os.ReadFile(path)
	require.NoErrorf(t, err, "failed to read file %s", path)

	assert.Equal(t, expected, string(actual))
}

func tmpDirWithCleanup(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "store")
	require.NoError(t, err, "failed to create temp dir")

	t.Cleanup(test.Cleanup(t, dir))

	return dir
}
