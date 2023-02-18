package mtgjson_test

import (
	"fmt"
	"os"
	"testing"

	logger "github.com/konstantinfoerster/card-importer-go/internal/log"
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
