package mtgjson_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/logger"
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
