package storage

import (
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	logger.SetupConsoleLogger()
	err := logger.SetLogLevel("warn")
	if err != nil {
		fmt.Printf("Failed to set log level %v", err)
		os.Exit(1)
	}

	exitVal := 0

	exitVal = m.Run()

	os.Exit(exitVal)
}

func TestStoredFileIsAlwaysInsideBasePath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	path := []string{"..", "dir", "..", "test.txt"}

	f, err := store.Store(strings.NewReader("content"), path...)
	if err != nil {
		t.Fatalf("failed to store in local storage, got: %v, wanted no error", err)
	}

	assert.Equal(t, filepath.Join(dir, "dir", "test.txt"), f.AbsolutePath)
}

func TestStoreWithSubDirs(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	path := []string{"dir", "sub", "sub2", "sub3", "test.txt"}

	f, err := store.Store(strings.NewReader("content"), path...)
	if err != nil {
		t.Fatalf("failed to store in local storage, got: %v, wanted no error", err)
	}

	assert.FileExists(t, f.AbsolutePath)
}

func TestStoreReturnsCorrectPath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	path := []string{"dir", "test.txt"}

	f, err := store.Store(strings.NewReader("content"), path...)
	if err != nil {
		t.Fatalf("failed to store in local storage, got: %v, wanted no error", err)
	}

	assert.Equal(t, filepath.Join(dir, "dir", "test.txt"), f.AbsolutePath)
	assert.Equal(t, filepath.Join("dir", "test.txt"), f.Path)
}

func TestStoreModeCreate(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	fileName := "test.txt"

	f, err := store.Store(strings.NewReader("content"), fileName)
	if err != nil {
		t.Fatalf("failed to store file, got: %v, wanted no error", err)
	}

	assert.FileExists(t, f.AbsolutePath)
	assertFileContent(t, "content", f.AbsolutePath)
}

func TestStoreModeCreateFails(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	fileName := "test.txt"

	_, err = store.Store(strings.NewReader("content"), fileName)
	if err != nil {
		t.Fatalf("failed to store file, got: %v, wanted no error", err)
	}
	_, err = store.Store(strings.NewReader("differentContent"), fileName)
	if err == nil {
		t.Fatalf("expected store to fail if file already exists")
	}

	assert.ErrorIs(t, err, os.ErrExist)
}

func TestStoreModeReplace(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.REPLACE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	fileName := "test.txt"

	_, err = store.Store(strings.NewReader("content"), fileName)
	if err != nil {
		t.Fatalf("failed to create file, got: %v, wanted no error", err)
	}
	f, err := store.Store(strings.NewReader("differentContent"), fileName)
	if err != nil {
		t.Fatalf("failed to replace file, got: %v, wanted no error", err)
	}

	assert.FileExists(t, f.AbsolutePath)
	assertFileContent(t, "differentContent", f.AbsolutePath)
}

func TestLoadNoneExistingFile(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	fileName := "notFound.txt"

	_, err = store.Load(fileName)
	if err == nil {
		t.Fatalf("expected a file not exists error but got no error")
	}

	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadWithoutAnyPath(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}

	_, err = store.Load("")
	if err == nil {
		t.Fatalf("expected not found error but got no error")
	}

	assert.Contains(t, err.Error(), "not supported")
}

func TestLoadFile(t *testing.T) {
	dir := tmpDirWithCleanup(t)
	store, err := NewLocalStorage(config.Storage{
		Location: dir,
		Mode:     config.CREATE,
	})
	if err != nil {
		t.Fatalf("failed to create local storage, got: %v, wanted no error", err)
	}
	fileName := "test.txt"
	expected := "content"
	_, err = store.Store(strings.NewReader(expected), fileName)
	if err != nil {
		t.Fatalf("failed to create file, got: %v, wanted no error", err)
	}

	cases := []struct {
		name string
		path []string
		want string
	}{
		{
			name: "LoadFile",
			path: []string{"test.txt"},
			want: "content",
		},
		{
			name: "LoadFileOutsideBasePathFallbackToBathPath",
			path: []string{"..", "..", "test.txt"},
			want: "content",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := store.Load(tc.path...)
			if err != nil {
				t.Fatalf("failed to load file, got: %v, wanted no error", err)
			}
			defer actual.Close()

			assertContentEquals(t, tc.want, actual)
		})
	}
}

func assertContentEquals(t *testing.T, expected string, r io.Reader) {
	actual, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("failed to read file %v", err)
		return
	}

	assert.Equal(t, expected, string(actual))
}

func assertFileContent(t *testing.T, expected string, path string) {
	actual, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read file %s %v", path, err)
		return
	}

	assert.Equal(t, expected, string(actual))
}

func tmpDirWithCleanup(t *testing.T) string {
	dir, err := ioutil.TempDir("", "store")
	if err != nil {
		t.Fatalf("failed to create temp dir %v", err)
	}
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
