package config_test

import (
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestDatabase_ConnectionURL(t *testing.T) {
	cfg := config.Database{
		Host:     "localhost",
		Port:     "5432",
		Database: "test",
		Username: "Tester",
		Password: "secret",
	}

	assert.Equal(t, "postgres://Tester:secret@localhost:5432/test", cfg.ConnectionURL())
}
