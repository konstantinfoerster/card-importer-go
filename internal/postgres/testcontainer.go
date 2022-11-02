package postgres

import (
	"context"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"path/filepath"
	"runtime"
	"testing"
)

func NewRunner() *DatabaseRunner {
	ctx := context.Background()
	return &DatabaseRunner{
		ctx: ctx,
	}
}

type DatabaseRunner struct {
	ctx  context.Context
	conn *DBConnection
}

func (r *DatabaseRunner) Run(t *testing.T, runTests func(t *testing.T)) {
	err := r.runPostgresContainer(func(cfg config.Database) error {
		conn, err := Connect(r.ctx, cfg)
		if err != nil {
			return err
		}
		defer func(toClose *DBConnection) {
			cErr := toClose.Close()
			if cErr != nil {
				// report close errors
				if err == nil {
					err = cErr
				} else {
					err = errors.Wrap(err, cErr.Error())
				}
			}
		}(conn)
		r.conn = conn

		runTests(t)

		return err
	})

	if err != nil {
		t.Fatalf("failed to start container %v", err)
	}
}

func (r *DatabaseRunner) Connection() *DBConnection {
	return r.conn
}

func (r *DatabaseRunner) Cleanup(t *testing.T) func() {
	return func() {
		cErr := r.conn.Cleanup()
		if cErr != nil {
			t.Fatalf("failed to cleanup database %v", cErr)
		}
	}
}

func (r *DatabaseRunner) runPostgresContainer(f func(c config.Database) error) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get caller")
	}
	dbDirLink := filepath.Join(filepath.Dir(file), "testdata", "db")
	dbDir, err := filepath.EvalSymlinks(dbDirLink)
	if err != nil {
		return err
	}
	username := "tester"
	password := "tester"
	database := "cardmanager"

	// TODO read env variables from config
	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(dbDir, "/docker-entrypoint-initdb.d"),
		),
		Env: map[string]string{
			"POSTGRES_DB":       "postgres",
			"POSTGRES_PASSWORD": "test",
			"APP_DB_USER":       username,
			"APP_DB_PASS":       password,
			"APP_DB_NAME":       database,
		},
		AlwaysPullImage: true,
		SkipReaper:      false,
		WaitingFor:      wait.ForLog("[1] LOG:  database system is ready to accept connections"),
	}

	postgresC, err := testcontainers.GenericContainer(r.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}
	defer func(toClose testcontainers.Container) {
		cErr := toClose.Terminate(r.ctx)
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Wrap(err, cErr.Error())
			}
		}
	}(postgresC)

	if log.Debug().Enabled() {
		logs, err := postgresC.Logs(r.ctx)
		if err != nil {
			return err
		}
		defer logs.Close()
		b, err := io.ReadAll(logs)
		if err != nil {
			return err
		}
		log.Debug().Msg(string(b))
	}

	ip, err := postgresC.Host(r.ctx)
	if err != nil {
		return err
	}

	mappedPort, err := postgresC.MappedPort(r.ctx, "5432")
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
	err = f(dbConfig)
	return err
}
