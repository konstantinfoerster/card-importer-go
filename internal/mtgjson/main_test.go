package mtgjson

import (
	"bufio"
	"encoding/json"
	"fmt"
	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
	"github.com/nsf/jsondiff"
	"io"
	"os"
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

func fromFile(t *testing.T, path string) io.Reader {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file %s", path)
	}

	return bufio.NewReader(f)
}

func assertEquals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()

	w, err := json.Marshal(expected)
	if err != nil {
		t.Errorf("failed to marshal want struct %v", err)
		return
	}
	g, err := json.Marshal(actual)
	if err != nil {
		t.Errorf("failed to marshal got struct %v", err)
		return
	}

	o := jsondiff.DefaultConsoleOptions()

	d, s := jsondiff.Compare(w, g, &o)

	if d != jsondiff.FullMatch {
		t.Errorf("found difference in struct, check result below:\n %v", s)
	}
}
