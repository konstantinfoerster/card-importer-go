package postgres

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type logConsumer struct{}

func (lc *logConsumer) Accept(l testcontainers.Log) {
	log.Debug().Msg(string(l.Content))
}

func NewRunner() *DatabaseRunner {
	return &DatabaseRunner{}
}

type DatabaseRunner struct {
	conn      *DBConnection
	container testcontainers.Container
}

func (r *DatabaseRunner) Connection() *DBConnection {
	return r.conn
}

func (r *DatabaseRunner) Start(ctx context.Context) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current dir")
	}

	dbDir, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(file), "testdata", "db"))
	if err != nil {
		return err
	}

	username := "tester"
	password := "tester"
	database := "cardmanager"

	var initScriptDirPermissions int64 = 0755
	// TODO: read env variables from config
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine3.20",
		ExposedPorts: []string{"5432/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(dbDir, "01-init.sh"),
				ContainerFilePath: "/docker-entrypoint-initdb.d/01-init.sh",
				FileMode:          initScriptDirPermissions,
			},
			{
				HostFilePath:      filepath.Join(dbDir, "02-create-tables.sql"),
				ContainerFilePath: "/docker-entrypoint-initdb.d/02-create-tables.sql",
				FileMode:          initScriptDirPermissions,
			},
		},
		Env: map[string]string{
			"POSTGRES_DB":       "postgres",
			"POSTGRES_PASSWORD": "test",
			"APP_DB_USER":       username,
			"APP_DB_PASS":       password,
			"APP_DB_NAME":       database,
		},
		AlwaysPullImage: true,
		WaitingFor:      wait.ForLog("[1] LOG:  database system is ready to accept connections"),
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts: []testcontainers.LogProductionOption{
				testcontainers.WithLogProductionTimeout(10 * time.Second),
			},
			Consumers: []testcontainers.LogConsumer{&logConsumer{}},
		},
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to create container %w", err)
	}

	r.container = postgresC

	ip, err := postgresC.Host(ctx)
	if err != nil {
		return err
	}

	mappedPort, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return err
	}

	dbConfig := config.Database{
		Username: username,
		Password: password,
		Host:     ip,
		Port:     mappedPort.Port(),
		Database: database,
	}

	conn, err := Connect(ctx, dbConfig)
	if err != nil {
		sErr := r.Stop(ctx)

		return errors.Join(err, sErr)
	}

	r.conn = conn

	return nil
}

func (r *DatabaseRunner) Stop(ctx context.Context) error {
	var err error
	if r.conn != nil {
		err = r.conn.Close()
	}

	if r.container != nil {
		cErr := r.container.Terminate(ctx)
		if err == nil {
			err = cErr
		} else {
			err = errors.Join(err, cErr)
		}
	}

	return err
}

func (r *DatabaseRunner) Cleanup(t *testing.T) func() {
	t.Helper()

	return func() {
		cErr := r.conn.Cleanup()
		if cErr != nil {
			t.Fatalf("failed to cleanup database %v", cErr)
		}
	}
}
