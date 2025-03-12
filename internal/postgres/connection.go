package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"github.com/rs/zerolog/log"
)

type DBConnection struct {
	Conn   DBConn
	pgxCon *pgxpool.Pool
}

func Connect(ctx context.Context, config config.Database) (*DBConnection, error) {
	c, err := pgxpool.ParseConfig(config.ConnectionURL())
	if err != nil {
		return nil, err
	}

	c.MaxConnLifetime = time.Second * time.Duration(5)
	c.MaxConnIdleTime = time.Millisecond * time.Duration(500)
	c.HealthCheckPeriod = time.Millisecond * time.Duration(500)
	c.MaxConns = config.MaxConnectionsOrDefault()

	log.Info().Msgf("max database connection is set to %d", c.MaxConns)

	pool, err := pgxpool.ConnectConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	dbConn := &DBConnection{
		Conn:   pool,
		pgxCon: pool,
	}

	return dbConn, err
}

func (d *DBConnection) Close() error {
	d.pgxCon.Close()

	return nil
}

func (d *DBConnection) WithTransaction(f func(conn *DBConnection) error) error {
	switch d.Conn.(type) {
	case pgx.Tx:
		return fmt.Errorf("already inside a transaction")
	default:
		opts := pgx.TxOptions{AccessMode: pgx.ReadWrite, IsoLevel: pgx.ReadCommitted}

		return d.pgxCon.BeginTxFunc(context.TODO(), opts, func(t pgx.Tx) error {
			dbCon := &DBConnection{
				Conn:   t,
				pgxCon: d.pgxCon,
			}

			return f(dbCon)
		})
	}
}

func (d *DBConnection) Cleanup() error {
	tables := []string{
		"card_set_translation",
		"card_set",

		"card_block_translation",
		"card_block",

		"face_super_type",
		"face_sub_type",
		"face_card_type",

		"card_translation",
		"card",
		"card_face",

		"card_type_translation",
		"card_type",
		"super_type_translation",
		"super_type",
		"sub_type_translation",
		"sub_type",

		"card_image",
	}
	_, err := d.Conn.Exec(context.TODO(), fmt.Sprintf("TRUNCATE %s RESTART IDENTITY", strings.Join(tables, ",")))

	return err
}

// DBConn Implemented by pgx.Conn and pgx.Tx.
type DBConn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}
