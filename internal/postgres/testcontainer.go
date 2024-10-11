package postgres

import (
	"context"
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewRunner() *DatabaseRunner {
	return &DatabaseRunner{}
}

type DatabaseRunner struct {
	conn *DBConnection
}

func (r *DatabaseRunner) Run(t *testing.T, runTests func(t *testing.T)) {
	t.Helper()

	ctx := context.Background()
	err := r.runPostgresContainer(ctx, func(cfg config.Database) error {
		conn, err := Connect(ctx, cfg)
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
	t.Helper()

	return func() {
		cErr := r.conn.Cleanup()
		if cErr != nil {
			t.Fatalf("failed to cleanup database %v", cErr)
		}
	}
}

func (r *DatabaseRunner) runPostgresContainer(ctx context.Context, f func(c config.Database) error) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current dir")
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
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}
	defer func(toClose testcontainers.Container) {
		cErr := toClose.Terminate(ctx)
		if cErr != nil {
			// report close errors
			if err == nil {
				err = cErr
			} else {
				err = errors.Wrap(err, cErr.Error())
			}
		}
	}(postgresC)

	if e := log.Debug(); e.Enabled() {
		logs, err := postgresC.Logs(ctx)
		if err != nil {
			return err
		}
		defer logs.Close()

		b, err := io.ReadAll(logs)
		if err != nil {
			return err
		}

		e.Msg(string(b))
	}

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
	err = f(dbConfig)

	return err
}
